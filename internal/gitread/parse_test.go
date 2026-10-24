package gitread

import "testing"

func TestParseHEAD(t *testing.T) {
	h, err := ParseHEAD([]byte("ref: refs/heads/main\n"))
	if err != nil || !h.Attached || h.Target != "refs/heads/main" {
		t.Fatalf("%+v %v", h, err)
	}
	h, err = ParseHEAD([]byte("3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a\n"))
	if err != nil || h.Attached || h.Hash == "" {
		t.Fatalf("%+v %v", h, err)
	}
	if _, err := ParseHEAD([]byte("not a head\n")); err == nil {
		t.Fatal("accepted garbage")
	}
	if _, err := ParseHEAD([]byte("ref: refs/heads/../x\n")); err == nil {
		t.Fatal("accepted traversal")
	}
}

func TestParsePackedRefs(t *testing.T) {
	const in = "# pack-refs with: peeled fully-peeled sorted\n" +
		"3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a refs/heads/main\n" +
		"^aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n" +
		"9b1e44c0aa52f3d7e6c1b8a9d0f2e3c4b5a69788 refs/heads/FETCH_HEAD\n"
	refs, err := ParsePackedRefs([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 2 {
		t.Fatalf("refs: %+v", refs)
	}
	if ShortBranch(refs[1].Name) != "FETCH_HEAD" {
		t.Fatal(refs[1])
	}
}

func TestParsePackedRefsGarbage(t *testing.T) {
	refs, err := ParsePackedRefs([]byte("this is not packed refs\n\x00\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 0 {
		t.Fatalf("%+v", refs)
	}
}
