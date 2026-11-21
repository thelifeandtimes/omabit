#!/usr/bin/env python3
"""Skill archive provenance/content checks; no installer or desktop access."""
import importlib.util
import json
import re
from pathlib import Path
import shutil
import tarfile
import tempfile
import unittest

ROOT=Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('packager',ROOT/'scripts/package-release.py')
packager=importlib.util.module_from_spec(spec);spec.loader.exec_module(packager)

class SkillPackageTests(unittest.TestCase):
    def test_each_skill_is_independently_link_complete(self):
        for name in packager.skill_names(ROOT):
            directory=(ROOT/'skills'/name).resolve()
            for path in directory.rglob('*.md'):
                for target in re.findall(r'\]\(([^)]+)\)',path.read_text()):
                    if '://' in target or target.startswith('#'):
                        continue
                    resolved=(path.parent/target.split('#',1)[0]).resolve()
                    self.assertTrue(resolved.is_relative_to(directory) and resolved.is_file(),str(path)+': '+target)

    def test_complete_source_bound_reproducible_archive(self):
        digests=[]
        with tempfile.TemporaryDirectory(prefix='omabit-skills-') as directory:
            root=Path(directory)
            for n in range(2):
                stage=root/str(n);stage.mkdir();assets=stage/'assets';assets.mkdir()
                metadata={'version':'0.3.0-edge.1','source_commit':'a'*40,'development':{'label':'fixture','snapshot_sha256':'b'*64,'modified':True}}
                entry=packager.package_skills(ROOT,stage,assets,metadata,1234567890)
                digests.append(entry['sha256'])
                with tarfile.open(assets/entry['name']) as archive:
                    self.assertTrue(all(m.isfile() or m.isdir() for m in archive))
                    prefix='omabit-skills-0.3.0-edge.1/'
                    receipt=json.load(archive.extractfile(prefix+'SKILLS.json'))
                    self.assertEqual(receipt['development'],metadata['development'])
                    self.assertEqual(receipt['source_commit'],metadata['source_commit'])
                    originals={p.relative_to(ROOT/'skills').as_posix():p for p in (ROOT/'skills').rglob('*.md')}
                    self.assertEqual(set(receipt['files']),{'skills/'+name for name in originals})
                    for name,path in originals.items():
                        self.assertEqual(archive.extractfile(prefix+'skills/'+name).read(),path.read_bytes())
                        self.assertEqual(receipt['files']['skills/'+name],packager.sha256(path))
                    # The checksum list covers every file except itself.
                    checks=archive.extractfile(prefix+'SHA256SUMS').read().decode().splitlines()
                    names={line.split('  ',1)[1] for line in checks}
                    self.assertEqual(names,set(receipt['files'])|{'SKILLS.json'})
            self.assertEqual(digests[0],digests[1])

    def test_binary_bundle_carries_complete_skills_and_cli(self):
        with tempfile.TemporaryDirectory(prefix='omabit-installed-docs-') as directory:
            target=Path(directory)/'bundle';target.mkdir()
            packager.package_documentation(ROOT,target)
            for relative in ('skills','docs/cli'):
                for original in (ROOT/relative).rglob('*.md'):
                    self.assertEqual((target/original.relative_to(ROOT)).read_bytes(),original.read_bytes())
            packager.checksums(target)
            checks=(target/'SHA256SUMS').read_text()
            self.assertIn('skills/omabit-operator/SKILL.md',checks)
            self.assertIn('skills/omabit-plugin-developer/references/interface-style.md',checks)
            self.assertIn('docs/cli/urbit-collector.md',checks)

    def test_historical_export_has_no_new_assets(self):
        with tempfile.TemporaryDirectory(prefix='omabit-skills-old-') as directory:
            root=Path(directory)
            self.assertIsNone(packager.package_skills(root,root,root,{},0))
            self.assertEqual(list(root.iterdir()),[])

    def test_incomplete_pair_is_refused(self):
        with tempfile.TemporaryDirectory(prefix='omabit-skills-incomplete-') as directory:
            root=Path(directory);(root/'skills/omabit-operator').mkdir(parents=True)
            with self.assertRaisesRegex(ValueError,'inventory'):
                packager.package_skills(root,root,root,{},0)

    def test_declared_third_skill_cannot_disappear(self):
        with tempfile.TemporaryDirectory(prefix='omabit-skills-missing-third-') as directory:
            root=Path(directory);shutil.copyfile(ROOT/'SKILLSET.json',root/'SKILLSET.json')
            for name in ('omabit-operator','omabit-plugin-developer'):
                shutil.copytree(ROOT/'skills'/name,root/'skills'/name)
            with self.assertRaisesRegex(ValueError,'inventory'):
                packager.skill_manifest(root,{'version':'fixture','source_commit':'a'*40})

    def test_link_or_executable_resource_is_not_packaged(self):
        for kind in ('link','executable'):
            with self.subTest(kind=kind),tempfile.TemporaryDirectory(prefix='omabit-skills-unsafe-') as directory:
                root=Path(directory);shutil.copytree(ROOT/'skills',root/'skills');shutil.copyfile(ROOT/'SKILLSET.json',root/'SKILLSET.json')
                resource=root/'skills/omabit-operator/references/unsafe.md'
                if kind=='link':resource.symlink_to('/etc/passwd')
                else:resource.write_text('not an executable package');resource.chmod(0o755)
                with self.assertRaises(ValueError):packager.package_skills(root,root,root,{},0)

if __name__=='__main__':unittest.main()
