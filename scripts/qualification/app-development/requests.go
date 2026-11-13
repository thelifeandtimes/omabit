// Emits only fixed fresh-fake test requests; never contacts a runtime.
package main

import (
	"encoding/json"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/deskwork"
	"os"
)

func main() {
	out := map[string]json.RawMessage{}
	save := func(name string, r deskwork.LensRequest, err error) {
		if err != nil {
			panic(err)
		}
		b, err := r.Body()
		if err != nil {
			panic(err)
		}
		out[name] = b
	}
	r, e := deskwork.CommitFiles("~zod", "omabit-test", deskwork.Observation{Exists: true, Revision: 3, ClayHash: "0v0", Publication: "private"}, []deskwork.DeltaFile{{Path: "hello.txt", Bytes: []byte("third source\n")}, {Path: "new.txt", Bytes: []byte("new source\n")}, {Path: "remove.txt", Delete: true}})
	save("commit-files", r, e)
	r, e = deskwork.MarkEquivalent("~zod", "omabit-test", 4, "hello.txt", []byte("third source\n"))
	save("committed-equal", r, e)
	r, e = deskwork.CommitFiles("~zod", "omabit-test", deskwork.Observation{Exists: true, Revision: 4, ClayHash: "0v0", Publication: "private"}, []deskwork.DeltaFile{{Path: "gen/broken.hoon", Bytes: []byte("|=  [broken\n")}})
	save("commit-invalid", r, e)
	r, e = deskwork.ClayFiles("~zod", "omabit-test", 2)
	save("clay-files", r, e)
	r, e = deskwork.NormalizeFiles("~zod", "omabit-test", 2, []deskwork.SourceFile{{Path: "hello.txt", Bytes: []byte("first source\n")}})
	save("normalized-files", r, e)
	r, e = deskwork.NormalizeFiles("~zod", "omabit-test", 2, []deskwork.SourceFile{{Path: "hello.txt", Bytes: []byte("different source\n")}})
	save("normalized-different", r, e)
	r, e = deskwork.ListDesks("~zod")
	save("desks", r, e)
	r, e = deskwork.FilePaths("~zod", "omabit-test", 2)
	save("files", r, e)
	r, e = deskwork.AutocommitActive("~zod")
	save("autocommit", r, e)
	r, e = deskwork.Inventory("~zod", "omabit-test")
	save("inventory", r, e)
	r, e = deskwork.Create("~zod", "omabit-test")
	save("create", r, e)
	r, e = deskwork.Mount("~zod", "omabit-test", "omabit-test", 0)
	save("mount", r, e)
	r, e = deskwork.Commit("omabit-test")
	save("commit", r, e)
	r, e = deskwork.PublishSource("omabit-test")
	save("publish-source", r, e)
	r, e = deskwork.MarkEquivalent("~zod", "omabit-test", 2, "hello.txt", []byte("first source\n"))
	save("equivalent", r, e)
	r, e = deskwork.MarkEquivalent("~zod", "omabit-test", 2, "hello.txt", []byte("different source\n"))
	save("different", r, e)
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
