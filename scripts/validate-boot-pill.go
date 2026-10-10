//go:build ignore

// Live, inert transport check. Uploads a small fixture artifact, never registers
// a ship and never asks Vere to execute the fixture.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"slices"
	"time"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/collection"
	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/protocol"
)

func main(){
	name:=flag.String("runner","","enrolled runner");expected:=flag.String("expected-runner-id","","pinned identity");flag.Parse()
	ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
	dir,err:=collection.DefaultDir();must(err);r,err:=collection.Open(dir);must(err);entry,err:=r.Get(ctx,*name);must(err)
	if entry.RunnerID!=*expected{panic("runner selection changed")}
	c:=entry.Client(true);defer c.Close();h,err:=c.Hello(ctx);must(err)
	if h.AccessRole!="operator"||!slices.Contains(h.Features,"key-boot-options"){panic("updated operator runner required")}
	before,err:=c.Snapshot(ctx);must(err)
	body:=bytes.Repeat([]byte("omabit-inert-pill-transfer-fixture\n"),2048)
	meta:=protocol.BootPill{SHA256:fmt.Sprintf("%x",sha256.Sum256(body)),Bytes:int64(len(body))}
	must(c.StageBootPill(ctx,bytes.NewReader(body),meta))
	after,err:=c.Snapshot(ctx);must(err)
	if len(before.Instances)!=len(after.Instances){panic("fleet changed during upload")}
	for i,a:=range after.Instances{b:=before.Instances[i];if a.ID!=b.ID||a.Ship!=b.Ship||a.Config!=b.Config||a.DesiredState!=b.DesiredState||a.ObservedState!=b.ObservedState||a.ActiveOperationID!=b.ActiveOperationID{panic("instance changed during inert upload")}}
	fmt.Printf("PASS: authenticated SSH upload verified %d inert bytes, SHA-256 %s; no ship registered or started.\n",meta.Bytes,meta.SHA256)
}

func must(err error){if err!=nil{panic(err)}}
