"""Inert host-entry refusals: no engine, subprocess, HTTP or state effects."""
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import Mock,patch
HERE=Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'fake-networks'))
import docker_case
import docker_guest
import docker_peer
import docker_runtime

class DockerEntry(unittest.TestCase):
    def denied(self,call):
        with patch.dict(os.environ,{},clear=True),patch.object(docker_runtime.os,'geteuid',return_value=1000),patch.object(docker_runtime.subprocess,'run') as run,patch.object(docker_case,'Engine') as case_engine,patch.object(docker_peer,'Engine') as peer_engine:
            with self.assertRaisesRegex(RuntimeError,'private keyed Docker guest admission required'):call()
            run.assert_not_called();case_engine.assert_not_called();peer_engine.assert_not_called()
    def test_guest_refuses_before_packages_or_services(self):
        with patch.object(docker_guest.os,'geteuid',return_value=1000),patch.object(docker_guest.subprocess,'run') as run:
            with self.assertRaisesRegex(RuntimeError,'wrong guest marker/uid'):docker_guest.main()
            run.assert_not_called()
    def test_case_refuses_before_engine(self):self.denied(lambda:docker_case.exercise({}))
    def test_peers_refuse_before_engine(self):self.denied(lambda:docker_peer.exercise({},'a'*32,'b'*64))
    def test_http_refuses_before_transport(self):
        engine=Mock();transport=docker_peer.Transport(engine,{})
        self.denied(lambda:transport.call('zod','/~/host'));engine.assert_not_called();engine.check_anchor.assert_not_called()
    def test_events_refuse_before_transport(self):
        engine=Mock();transport=docker_peer.Transport(engine,{})
        self.denied(lambda:transport.events('/~/channel/omabit-state-'+'a'*32,None,{},True));engine.check_anchor.assert_not_called()

if __name__=='__main__':unittest.main()
