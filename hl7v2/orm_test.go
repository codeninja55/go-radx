package hl7v2

import "testing"

func TestAsORM(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORM))
	if _, ok := AsORM(msg); !ok {
		t.Fatal("AsORM(canonical ORM) = false, want true")
	}

	// A non-ORM message type is not an ORM.
	oru, _ := Parse([]byte("MSH|^~\\&|A|B|C|D|202605311230||ORU^R01|M1|P|2.4\r"))
	if _, ok := AsORM(oru); ok {
		t.Error("AsORM(ORU) = true, want false")
	}
}

func TestOrdersOneGroup(t *testing.T) {
	msg, _ := Parse([]byte(canonicalORM))
	orm, ok := AsORM(msg)
	if !ok {
		t.Fatal("AsORM = false")
	}

	var groups []OrderGroup
	for g := range orm.Orders() {
		groups = append(groups, g)
	}
	if len(groups) != 1 {
		t.Fatalf("Orders() yielded %d groups, want 1", len(groups))
	}
	g := groups[0]
	if g.Common.OrderControl != "NW" {
		t.Errorf("group.Common.OrderControl = %q, want NW", g.Common.OrderControl)
	}
	if len(g.Requests) != 1 {
		t.Fatalf("group.Requests = %d, want 1", len(g.Requests))
	}
	if g.Requests[0].UniversalServiceID.Code != "36643-5" {
		t.Errorf("group OBR-4 = %q, want 36643-5", g.Requests[0].UniversalServiceID.Code)
	}
}

func TestOrdersMultipleGroups(t *testing.T) {
	// Two ORC+OBR pairs: each ORC opens a group; the trailing OBR joins the
	// second ORC's group.
	body := "MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r" +
		"ORC|NW|P1|F1\r" +
		"OBR|1|P1|F1|AAA^A^LN\r" +
		"ORC|NW|P2|F2\r" +
		"OBR|1|P2|F2|BBB^B^LN\r" +
		"OBR|2|P2|F2|CCC^C^LN\r"
	msg, _ := Parse([]byte(body))
	orm, _ := AsORM(msg)

	var groups []OrderGroup
	for g := range orm.Orders() {
		groups = append(groups, g)
	}
	if len(groups) != 2 {
		t.Fatalf("Orders() yielded %d groups, want 2", len(groups))
	}
	if len(groups[0].Requests) != 1 {
		t.Errorf("group[0].Requests = %d, want 1", len(groups[0].Requests))
	}
	if len(groups[1].Requests) != 2 {
		t.Errorf("group[1].Requests = %d, want 2", len(groups[1].Requests))
	}
}

func TestOrdersEarlyBreak(t *testing.T) {
	// The iterator must honour a consumer that stops early.
	body := "MSH|^~\\&|A|B|C|D|202605311230||ORM^O01|M1|P|2.4\r" +
		"ORC|NW|P1|F1\r" +
		"ORC|NW|P2|F2\r"
	msg, _ := Parse([]byte(body))
	orm, _ := AsORM(msg)

	count := 0
	for range orm.Orders() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("early break consumed %d groups, want 1", count)
	}
}
