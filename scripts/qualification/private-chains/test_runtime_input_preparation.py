"""Inert preparation boundaries; no Docker, dependency process or database open."""
import io
import json
import os
from pathlib import Path
import stat
import tempfile
from types import SimpleNamespace
import unittest

import docker_appliance
import prepare_runtime_inputs as prepare


class RuntimeInputPreparationTests(unittest.TestCase):
    def archive(self, entries):
        out=io.BytesIO()
        for i,(name,data,mode) in enumerate(entries,1):docker_appliance.write_cpio(out,name,data,mode,i)
        docker_appliance.write_cpio(out,'TRAILER!!!',b'',0,len(entries)+1)
        return out.getvalue()

    def test_extracts_only_reviewed_payload_and_rejects_replacement(self):
        raw=self.archive([('init',b'not executed',stat.S_IFREG|0o755),('payload.tar.gz',b'payload',stat.S_IFREG|0o600)])
        self.assertEqual(prepare.payload_from_cpio(raw,prepare.sha(b'payload')),b'payload')
        for expected in ['a'*64,'bad']:
            with self.subTest(expected=expected),self.assertRaises(ValueError):prepare.payload_from_cpio(raw,expected)

    def test_ambiguous_special_missing_and_truncated_payloads_refuse(self):
        variants=[self.archive([('payload.tar.gz',b'p',stat.S_IFLNK|0o777)]),self.archive([('payload.tar.gz',b'p',stat.S_IFREG|0o600)]*2),self.archive([]),b'070701',self.archive([('payload.tar.gz',b'p',stat.S_IFREG|0o600)])[:-40]]
        for n,raw in enumerate(variants):
            with self.subTest(n=n),self.assertRaises(ValueError):prepare.payload_from_cpio(raw,prepare.sha(b'p'))

    def test_inventory_refuses_duplicate_oversize_and_noncanonical_entries(self):
        row={'bytes':1,'sha256':'a'*64}
        self.assertEqual(prepare.inventory(json.dumps({'vere':row}).encode()),{'vere':row})
        variants=[b'{"vere":{"bytes":1,"sha256":"'+b'a'*64+b'"},"vere":{}}',json.dumps({'../escape':row}).encode(),json.dumps({'tools/a':dict(row,bytes=True)}).encode(),json.dumps({'tools/a':dict(row,bytes=prepare.MAX_BYTES+1)}).encode(),json.dumps({'tools/a':dict(row,extra='unknown')}).encode()]
        for n,raw in enumerate(variants):
            with self.subTest(n=n),self.assertRaises(ValueError):prepare.inventory(raw)

    def test_noncanonical_paths_and_nonhex_header_fields_refuse(self):
        for name in ['.', '', '../x', '/x', 'tools/./x', 'tools//x', 'tools/x\\y', 'tools/x\n']:
            with self.subTest(name=name),self.assertRaises(ValueError):prepare.relative(name)
        raw=self.archive([('payload.tar.gz',b'p',stat.S_IFREG|0o600)])
        raw=raw[:54]+b'-0000001'+raw[62:]
        with self.assertRaises(ValueError):prepare.payload_from_cpio(raw,prepare.sha(b'p'))

    def test_selection_excludes_peers_keys_and_runtime_outputs(self):
        names=['src/node-closure.json','src/docker_gateway.cjs','src/docker_gateway_resume.cjs','src/gateway_resume_contract.cjs','vere','tools/package-lock.json','compiled/result.json','deployment/chain/CURRENT','source/bridge/package.json','src/docker_gateway_peers.py','secrets/zod.key','piers/zod/data','results/session.cookie']
        selected=prepare.selection(dict.fromkeys(names))
        for name in names[-4:]:self.assertNotIn(name,selected)
        self.assertEqual(selected['src/docker_gateway.cjs'],'gateway/docker_gateway.cjs')
        with self.assertRaises(ValueError):prepare.selection({})

    def test_file_reader_refuses_links_special_files_bounds_and_hash_changes(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);p=root/'file';p.write_bytes(b'content')
            self.assertEqual(prepare.read_file(p,8,prepare.sha(b'content')),b'content')
            hard=root/'hard';os.link(p,hard)
            with self.assertRaises(ValueError):prepare.read_file(p,8)
            hard.unlink();link=root/'link';link.symlink_to(p)
            with self.assertRaises(OSError):prepare.read_file(link,8)
            fifo=root/'fifo';os.mkfifo(fifo)
            for path,limit,expected in [(fifo,8,None),(root,8,None),(p,1,None),(p,8,'b'*64)]:
                before=set(Path('/proc/self/fd').iterdir())
                with self.subTest(path=path.name,limit=limit),self.assertRaises(ValueError):prepare.read_file(path,limit,expected)
                self.assertEqual(set(Path('/proc/self/fd').iterdir()),before)

    def test_existing_output_is_preserved_and_public_parent_refused(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);out=root/'output';out.mkdir();(out/'sentinel').write_bytes(b'preserve')
            with self.assertRaises(ValueError):prepare.prepare(SimpleNamespace(output=out))
            self.assertEqual((out/'sentinel').read_bytes(),b'preserve')
            root.chmod(0o755)
            with self.assertRaises(ValueError):prepare.output_path(root/'new')
            root.chmod(0o700)

    def test_wrong_build_review_refuses_before_output_creation(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);appliance=root/'appliance';appliance.mkdir();(appliance/'build.json').write_text('{}')
            out=root/'new'
            with self.assertRaises(ValueError):prepare.prepare(SimpleNamespace(output=out,appliance=appliance,build_sha256='a'*64))
            self.assertFalse(out.exists())


if __name__=='__main__':unittest.main()
