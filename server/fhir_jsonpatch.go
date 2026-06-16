package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/codeninja55/go-radx/fhir"
)

// JSON Patch (RFC 6902) applied to a FHIR resource for the server role's PATCH interaction. The
// patch is applied to the resource's JSON document, the result re-decoded into the release's
// concrete type, and the role then re-validates it through the release validator (the same gate
// create uses). A self-contained RFC 6902 applier is used rather than a dependency: the operation
// set is small and the FHIR scope is JSON Patch over a single resource document.
//
// FHIRPath Patch (a FHIR Parameters body, http.html#patch) is out of scope and documented as such;
// only application/json-patch+json is accepted.

// errPatch wraps a JSON Patch application failure with a PHI-free diagnostic. The path and op are
// structural locators (a JSON Pointer, an op name), never a patient value, so naming them is safe
// (PRD §9.1).
var errPatch = errors.New("json patch")

// applyJSONPatch applies an RFC 6902 JSON Patch document to a resource and returns the patched,
// re-decoded release resource. On a malformed patch document, an operation that does not apply (a
// bad path, a failed test), or a result that no longer decodes as a FHIR resource, it returns a nil
// resource with the HTTP status and a PHI-free diagnostic the caller writes as an OperationOutcome.
// The patched document is re-decoded through the release adapter so the returned resource is the
// release's concrete type, ready for validation and storage. A patch that changes the resourceType
// is rejected — a PATCH must not retype the resource at its id.
func (h *fhirHandler) applyJSONPatch(current fhir.Resource, patchDoc []byte) (fhir.Resource, int, string) {
	currentJSON, err := json.Marshal(current)
	if err != nil {
		return nil, http.StatusInternalServerError, "the current resource could not be encoded for patching"
	}
	patched, err := applyRFC6902(currentJSON, patchDoc)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, sanitizePatchError(err)
	}
	resource, err := h.adapter.unmarshalResource(patched)
	if err != nil {
		return nil, http.StatusUnprocessableEntity, "the patched resource is not a valid FHIR resource"
	}
	if resource.ResourceType() != current.ResourceType() {
		return nil, http.StatusUnprocessableEntity,
			"a patch must not change the resourceType from " + current.ResourceType() + " to " + resource.ResourceType()
	}
	return resource, 0, ""
}

// patchOp is one RFC 6902 operation. Path and From are JSON Pointers (RFC 6901); Value is the
// operand for add/replace/test. The raw value is kept as json.RawMessage so it is spliced into the
// document verbatim, preserving number lexical form and key order within the operand. The hasPath /
// hasFrom / hasValue flags record whether each member was present in the operation object, so an
// omitted member is distinguished from one that is present but the empty string "" (the valid
// whole-document JSON Pointer). This distinction is what lets validateOp reject an operation that
// is missing a required member (RFC 6902 §4) rather than silently treating the missing pointer as
// the document root.
type patchOp struct {
	Op    string
	Path  string
	From  string
	Value json.RawMessage

	hasPath  bool
	hasFrom  bool
	hasValue bool
}

// patchOpWire is the on-the-wire shape of an operation. Path and From are pointers so an absent
// member decodes to nil (distinguishable from a present "") and Value is a json.RawMessage so a
// present-but-null value is still detectable. UnmarshalJSON funnels through this to set the
// presence flags on patchOp.
type patchOpWire struct {
	Op    string          `json:"op"`
	Path  *string         `json:"path"`
	From  *string         `json:"from"`
	Value json.RawMessage `json:"value"`
}

// UnmarshalJSON decodes one operation, recording which optional members were present so validateOp
// can enforce RFC 6902's per-op required-member rules. A member that is present but empty ("") is
// recorded as present, so the legitimate whole-document pointer survives, while an omitted member
// stays absent.
func (op *patchOp) UnmarshalJSON(data []byte) error {
	var wire patchOpWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	op.Op = wire.Op
	op.Value = wire.Value
	op.hasValue = wire.Value != nil
	if wire.Path != nil {
		op.Path = *wire.Path
		op.hasPath = true
	}
	if wire.From != nil {
		op.From = *wire.From
		op.hasFrom = true
	}
	return nil
}

// validateOp enforces RFC 6902 §4's required-member rules for one operation before it is applied:
// add/replace/test require path and value, remove requires path, move/copy require path and from. A
// missing required member is rejected here (mapping to 422) rather than defaulting to the empty
// pointer "" (the document root), which would let, for example, a copy with no from silently read
// the whole document. An unknown op is left to applyOp, which rejects it.
func validateOp(op *patchOp) error {
	switch op.Op {
	case "add", "replace", "test":
		if !op.hasPath {
			return fmt.Errorf("%w: %s requires a path member", errPatch, op.Op)
		}
		if !op.hasValue {
			return fmt.Errorf("%w: %s requires a value member", errPatch, op.Op)
		}
	case "remove":
		if !op.hasPath {
			return fmt.Errorf("%w: remove requires a path member", errPatch)
		}
	case "move", "copy":
		if !op.hasPath {
			return fmt.Errorf("%w: %s requires a path member", errPatch, op.Op)
		}
		if !op.hasFrom {
			return fmt.Errorf("%w: %s requires a from member", errPatch, op.Op)
		}
	}
	return nil
}

// applyRFC6902 applies a JSON Patch document (a JSON array of operations) to a JSON document and
// returns the patched document. Operations are applied in order; the first failing operation aborts
// the whole patch (RFC 6902 §5: a patch is applied atomically — if any operation fails, the target
// is unchanged), so the caller never stores a half-applied result. The supported operations are the
// full RFC 6902 set: add, remove, replace, move, copy, test.
func applyRFC6902(docJSON, patchJSON []byte) ([]byte, error) {
	var ops []patchOp
	if err := json.Unmarshal(patchJSON, &ops); err != nil {
		return nil, fmt.Errorf("%w: the patch document is not a JSON Patch array", errPatch)
	}
	if len(ops) == 0 {
		return nil, fmt.Errorf("%w: the patch document has no operations", errPatch)
	}
	doc, err := decodeJSONPreservingNumbers(docJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: the target document is not valid JSON", errPatch)
	}
	for i := range ops {
		if err := validateOp(&ops[i]); err != nil {
			return nil, err
		}
		next, err := applyOp(doc, &ops[i])
		if err != nil {
			return nil, err
		}
		doc = next
	}
	return json.Marshal(doc)
}

// decodeJSONPreservingNumbers decodes JSON into the any tree the patch operates on with
// UseNumber, so every JSON number stays a json.Number (its exact lexical text) rather than being
// coerced to float64. This is what keeps a patch to one field from silently rewriting FHIR decimal
// and integer64 values elsewhere in the document: a json.Number round-trips byte-for-byte on the
// final json.Marshal (1.00 stays 1.00, a 64-bit integer keeps full precision), whereas float64 would
// drop trailing zeros and lose precision past 2^53. Only the patched path is changed; every untouched
// number is re-emitted verbatim.
func decodeJSONPreservingNumbers(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// applyOp applies one operation to doc and returns the resulting document root (the root may itself
// change for an op whose path is "", i.e. the whole document). It dispatches on the op name; an
// unknown op is an error so a malformed patch never silently no-ops.
func applyOp(doc any, op *patchOp) (any, error) {
	switch op.Op {
	case "add":
		return applyAddReplace(doc, op, false)
	case "replace":
		return applyAddReplace(doc, op, true)
	case "remove":
		_, root, err := removeAt(doc, op.Path)
		return root, err
	case "test":
		return doc, applyTest(doc, op)
	case "move":
		return applyMoveCopy(doc, op, true)
	case "copy":
		return applyMoveCopy(doc, op, false)
	default:
		return nil, fmt.Errorf("%w: unsupported operation %q", errPatch, op.Op)
	}
}

// applyAddReplace performs an add or replace at op.Path. add inserts (or overwrites a member, or
// inserts into an array at the index/`-`); replace requires the target to already exist (RFC 6902
// §4.3). The operand is decoded from op.Value, so it is spliced as a parsed value.
func applyAddReplace(doc any, op *patchOp, mustExist bool) (any, error) {
	if op.Value == nil {
		return nil, fmt.Errorf("%w: %s at %q requires a value", errPatch, op.Op, op.Path)
	}
	value, err := decodeJSONPreservingNumbers(op.Value)
	if err != nil {
		return nil, fmt.Errorf("%w: %s value is not valid JSON", errPatch, op.Op)
	}
	if op.Path == "" {
		// Replacing or adding at the root replaces the whole document.
		return value, nil
	}
	if mustExist {
		if _, err := getAt(doc, op.Path); err != nil {
			return nil, err
		}
	}
	return setAt(doc, op.Path, value)
}

// applyTest evaluates a test operation: the value at op.Path must deep-equal op.Value, else the
// patch fails (RFC 6902 §4.6). Equality is structural (via canonical JSON), so key order and
// whitespace in the operand do not matter.
func applyTest(doc any, op *patchOp) error {
	got, err := getAt(doc, op.Path)
	if err != nil {
		return err
	}
	want, err := decodeJSONPreservingNumbers(op.Value)
	if err != nil {
		return fmt.Errorf("%w: test value is not valid JSON", errPatch)
	}
	gotJSON, err1 := json.Marshal(got)
	wantJSON, err2 := json.Marshal(want)
	if err1 != nil || err2 != nil || string(gotJSON) != string(wantJSON) {
		return fmt.Errorf("%w: test at %q failed", errPatch, op.Path)
	}
	return nil
}

// applyMoveCopy performs a move or copy: the value at op.From is read, then (for move) removed, then
// added at op.Path. A move into one of its own children is rejected (RFC 6902 §4.4).
func applyMoveCopy(doc any, op *patchOp, isMove bool) (any, error) {
	value, err := getAt(doc, op.From)
	if err != nil {
		return nil, err
	}
	if isMove {
		if op.Path == op.From || strings.HasPrefix(op.Path, op.From+"/") {
			return nil, fmt.Errorf("%w: move from %q into its own location is invalid", errPatch, op.From)
		}
		_, root, rerr := removeAt(doc, op.From)
		if rerr != nil {
			return nil, rerr
		}
		doc = root
	}
	// Re-encode/decode the value so move/copy splice an independent copy, not a shared reference.
	// The decode preserves number lexicals (UseNumber) exactly as add/replace and the untouched
	// document do: a plain json.Unmarshal would coerce every number in the copied subtree to float64,
	// rewriting a FHIR decimal (1.00 -> 1) or losing int64 precision past 2^53. Because the source was
	// decoded with UseNumber, json.Marshal here re-emits json.Number values verbatim, and UseNumber on
	// the way back keeps them lexical, so the copied subtree round-trips byte-for-byte.
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: could not copy the value at %q", errPatch, op.From)
	}
	fresh, err := decodeJSONPreservingNumbers(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: could not copy the value at %q", errPatch, op.From)
	}
	if op.Path == "" {
		return fresh, nil
	}
	return setAt(doc, op.Path, fresh)
}

// jsonPointerTokens splits a JSON Pointer (RFC 6901) into its decoded reference tokens. The leading
// "/" is required for a non-root pointer; each token has "~1" unescaped to "/" and "~0" to "~". An
// empty pointer ("") yields no tokens (the whole document), handled by the callers.
func jsonPointerTokens(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("%w: %q is not a valid JSON Pointer", errPatch, pointer)
	}
	parts := strings.Split(pointer[1:], "/")
	for i, p := range parts {
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		parts[i] = p
	}
	return parts, nil
}

// getAt returns the value at the JSON Pointer path within doc, or an error when the path does not
// resolve. The empty path returns the whole document.
func getAt(doc any, path string) (any, error) {
	tokens, err := jsonPointerTokens(path)
	if err != nil {
		return nil, err
	}
	cur := doc
	for _, tok := range tokens {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[tok]
			if !ok {
				return nil, fmt.Errorf("%w: path %q does not exist", errPatch, path)
			}
			cur = v
		case []any:
			idx, err := arrayIndex(tok, len(node), false)
			if err != nil {
				return nil, err
			}
			cur = node[idx]
		default:
			return nil, fmt.Errorf("%w: path %q does not exist", errPatch, path)
		}
	}
	return cur, nil
}

// setAt sets value at the JSON Pointer path within doc and returns the (possibly new) document root.
// It walks to the parent of the final token, then inserts: into an object it sets the member; into an
// array it inserts at the index (or appends at "-"). A missing intermediate node is an error — RFC
// 6902 add does not create intermediate containers.
func setAt(doc any, path string, value any) (any, error) {
	tokens, err := jsonPointerTokens(path)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return value, nil
	}
	parent, err := getAt(doc, parentPointer(path))
	if err != nil {
		return nil, err
	}
	last := tokens[len(tokens)-1]
	switch node := parent.(type) {
	case map[string]any:
		node[last] = value
	case []any:
		newArr, err := insertIntoArray(node, last, value)
		if err != nil {
			return nil, err
		}
		// Re-attach the grown array to its parent: a slice append may reallocate, so the parent's
		// reference must be updated. The root is returned by re-setting the parent path.
		if len(tokens) == 1 {
			return newArr, nil
		}
		return setAt(doc, parentPointer(path), newArr)
	default:
		return nil, fmt.Errorf("%w: path %q parent is not a container", errPatch, path)
	}
	return doc, nil
}

// removeAt removes the value at the JSON Pointer path within doc and returns the removed value and
// the (possibly new) document root. Removing from an object deletes the member; from an array it
// drops the element and re-slices. A missing target is an error.
func removeAt(doc any, path string) (removed any, root any, err error) {
	tokens, terr := jsonPointerTokens(path)
	if terr != nil {
		return nil, nil, terr
	}
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("%w: cannot remove the whole document", errPatch)
	}
	parent, perr := getAt(doc, parentPointer(path))
	if perr != nil {
		return nil, nil, perr
	}
	last := tokens[len(tokens)-1]
	switch node := parent.(type) {
	case map[string]any:
		v, ok := node[last]
		if !ok {
			return nil, nil, fmt.Errorf("%w: path %q does not exist", errPatch, path)
		}
		delete(node, last)
		return v, doc, nil
	case []any:
		idx, ierr := arrayIndex(last, len(node), false)
		if ierr != nil {
			return nil, nil, ierr
		}
		v := node[idx]
		newArr := append(node[:idx:idx], node[idx+1:]...)
		if len(tokens) == 1 {
			return v, newArr, nil
		}
		newRoot, serr := setAt(doc, parentPointer(path), newArr)
		return v, newRoot, serr
	default:
		return nil, nil, fmt.Errorf("%w: path %q parent is not a container", errPatch, path)
	}
}

// insertIntoArray inserts value into arr at the token position: a numeric index inserts before the
// existing element at that index (RFC 6902 array add), and "-" appends. An out-of-range index is an
// error.
func insertIntoArray(arr []any, token string, value any) ([]any, error) {
	if token == "-" {
		return append(arr, value), nil
	}
	idx, err := arrayIndex(token, len(arr), true)
	if err != nil {
		return nil, err
	}
	arr = append(arr, nil)
	copy(arr[idx+1:], arr[idx:])
	arr[idx] = value
	return arr, nil
}

// arrayIndex parses an array index token. forInsert allows the one-past-the-end index (an add may
// insert at len), while a get/remove index must address an existing element. A non-numeric or
// out-of-range token is an error. A leading-zero multi-digit token is rejected per RFC 6901.
func arrayIndex(token string, length int, forInsert bool) (int, error) {
	if token == "" || (len(token) > 1 && token[0] == '0') {
		return 0, fmt.Errorf("%w: %q is not a valid array index", errPatch, token)
	}
	idx, err := strconv.Atoi(token)
	if err != nil || idx < 0 {
		return 0, fmt.Errorf("%w: %q is not a valid array index", errPatch, token)
	}
	limit := length
	if !forInsert {
		limit = length - 1
	}
	if idx > limit {
		return 0, fmt.Errorf("%w: array index %d is out of range", errPatch, idx)
	}
	return idx, nil
}

// parentPointer returns the JSON Pointer of the parent of path (path with its last token dropped).
func parentPointer(path string) string {
	i := strings.LastIndex(path, "/")
	if i <= 0 {
		return ""
	}
	return path[:i]
}

// sanitizePatchError renders a JSON Patch failure as a PHI-free diagnostic. The errPatch messages
// are built from op names, JSON Pointers, and indices — structural locators, never patient values —
// so the message is surfaced as-is; a non-errPatch error is reduced to a generic message so an
// unexpected error never leaks an internal detail.
func sanitizePatchError(err error) string {
	if errors.Is(err, errPatch) {
		return err.Error()
	}
	return "the patch could not be applied"
}
