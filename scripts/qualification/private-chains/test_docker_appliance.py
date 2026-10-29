"""Inert preparation checks: no VM, Docker, ships or actual Go invocation."""
import argparse
import json
import os
from pathlib import Path
import stat
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import docker_appliance as prep

class PreparationTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(prefix='omabit-preparation-check-');self.addCleanup(self.tmp.cleanup)
        self.root=Path(self.tmp.name);self.inputs=self.root/'inputs';self.inputs.mkdir()
        (self.inputs/'a').write_bytes(b'input')
    def test_seal_deterministic_and_internal_links_are_flattened(self):
        (self.inputs/'alias').symlink_to('a')
        one=self.root/'one.gz';two=self.root/'two.gz'
        first=prep.seal_payload(one,[(self.inputs,'tools',self.inputs)])
        second=prep.seal_payload(two,[(self.inputs,'tools',self.inputs)])
        self.assertEqual(first,second);self.assertEqual(one.read_bytes(),two.read_bytes())
        self.assertEqual(set(first),{'tools/a','tools/alias'})
        with tarfile.open(one) as tar:self.assertTrue(all(m.isfile() for m in tar))
    def test_escape_cycle_and_special_refused(self):
        for kind in ('escape','cycle','fifo'):
            with self.subTest(kind=kind):
                target=self.inputs/'unsafe'
                if kind=='escape':target.symlink_to(self.root/'outside');(self.root/'outside').write_bytes(b'private')
                elif kind=='cycle':target.symlink_to(self.inputs,target_is_directory=True)
                else:os.mkfifo(target)
                try:
                    with self.assertRaises(ValueError):prep.seal_payload(self.root/(kind+'.gz'),[(self.inputs,'tools',self.inputs)])
                finally:target.unlink()
    def test_path_and_input_hash_refusal(self):
        for name in ('','.', '../a','/a','a/../b','a//b','a\\b'):
            with self.subTest(name=name),self.assertRaises(ValueError):prep.relative(name)
        with self.assertRaises(ValueError):prep.checked_file(self.inputs,'a','0'*64)
        (self.inputs/'link').symlink_to('/dev/null')
        with self.assertRaises(ValueError):prep.checked_file(self.inputs,'link')
    def test_duplicate_and_size_limits(self):
        item=(self.inputs/'a','tools/a',self.inputs)
        with self.assertRaises(ValueError):prep.seal_payload(self.root/'dup.gz',[item,item])
        with patch.object(prep,'MAX_BYTES',4),self.assertRaises(ValueError):prep.seal_payload(self.root/'large.gz',[item])
        with patch.object(prep,'MAX_FILES',0),self.assertRaises(ValueError):prep.seal_payload(self.root/'count.gz',[item])
    def test_changed_sealed_bytes_refused(self):
        path=self.root/'sealed.gz';inventory=prep.seal_payload(path,[(self.inputs,'tools',self.inputs)])
        inventory['tools/a']['sha256']='0'*64
        with self.assertRaises(ValueError):prep.verify_payload(path,inventory)
    def archive(self,entries):
        path=self.root/'initramfs.cpio'
        with path.open('wb') as stream:
            for i,(name,data,mode) in enumerate(entries):prep.write_cpio(stream,name,data,mode,i+1)
            prep.write_cpio(stream,'TRAILER!!!',b'',0,len(entries)+1)
        return path
    def test_cpio_roundtrip(self):
        entries=[('init',b'ELF',stat.S_IFREG|0o755),('modules',b'',stat.S_IFDIR|0o755)]
        self.assertEqual(prep.cpio_entries(self.archive(entries)),entries)
    def test_cpio_refusals(self):
        init=('init',b'ELF',stat.S_IFREG|0o755)
        for extra in (init,('../escape',b'',stat.S_IFREG),('payload.tar.gz',b'',stat.S_IFREG),('link',b'init',stat.S_IFLNK)):
            with self.subTest(name=extra[0]),self.assertRaises(ValueError):prep.cpio_entries(self.archive([init,extra]))
        with self.assertRaises(ValueError):prep.cpio_entries(self.archive([]))
        path=self.archive([init]);path.write_bytes(path.read_bytes()[:30])
        with self.assertRaises(ValueError):prep.cpio_entries(path)
    def test_validation_failure_has_no_effect(self):
        with patch.object(prep,'validate_inputs',side_effect=ValueError('changed source')),patch.object(prep.subprocess,'run') as run,patch.object(prep.storage,'admit') as admit:
            with self.assertRaisesRegex(ValueError,'changed source'):prep.prepare(object())
            run.assert_not_called();admit.assert_not_called()
    def test_output_refusal_before_reading_inputs(self):
        args=argparse.Namespace(**{key:self.inputs for key in
            ('base_appliance','tools','sources','compiled','deployment','node_runtime')})
        occupied=self.root/'existing';occupied.mkdir()
        indirect=self.root/'indirect';indirect.symlink_to(self.root/'missing')
        for output in (occupied,indirect,self.inputs/'recursive'):
            with self.subTest(output=output.name),patch.object(prep,'read_json') as read:
                args.output=output
                with self.assertRaises(ValueError):prep.validate_inputs(args)
                read.assert_not_called()

    def test_metadata_bound(self):
        path=self.root/'json';path.write_text('{"ok":true}')
        with self.assertRaises(ValueError):prep.read_json(path,limit=2)
        self.assertEqual(prep.read_json(path),{'ok':True})

if __name__=='__main__':unittest.main()
