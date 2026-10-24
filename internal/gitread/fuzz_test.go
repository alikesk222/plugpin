package gitread

import "testing"

func FuzzParseHEAD(f *testing.F) {
	f.Add([]byte("ref: refs/heads/main\n"))
	f.Add([]byte("3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a\n"))
	f.Add([]byte(""))
	f.Add([]byte("ref: refs/heads/FETCH_HEAD\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		h, err := ParseHEAD(data)
		if err != nil {
			return
		}
		if h.Attached && ShortBranch(h.Target) == "" && h.Target != "" {
			if len(h.Target) > 512 {
				t.Fatalf("huge target accepted")
			}
		}
		if !h.Attached && h.Hash != "" && !hexHash(h.Hash) {
			t.Fatalf("bad hash %q", h.Hash)
		}
	})
}

func FuzzParsePackedRefs(f *testing.F) {
	f.Add([]byte("# comment\nabc refs/heads/main\n"))
	f.Add([]byte("3f9c2a7d4e1b0c8a9f6e5d4c3b2a1f0e9d8c7b6a refs/heads/main\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		refs, err := ParsePackedRefs(data)
		if err != nil {
			return
		}
		for _, ref := range refs {
			if !hexHash(ref.Hash) || ShortBranch(ref.Name) == "" && !validRef(ref.Name) {
				t.Fatalf("bad ref %+v", ref)
			}
		}
	})
}
