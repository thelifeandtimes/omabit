"""No runtime operations: check host refusal and the actual boot/reopen argv."""
import importlib.util
import os
from pathlib import Path
import sys
import unittest
from unittest.mock import patch
HERE=Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent/'fake-networks'))
spec=importlib.util.spec_from_file_location('private_native_peer',HERE/'native_peer.py')
peer=importlib.util.module_from_spec(spec);spec.loader.exec_module(peer)

class NativeEntry(unittest.TestCase):
    def test_host_refuses_before_process_or_http(self):
        for environment in ({},{'OMABIT_PRIVATE_CHAIN':'private-chain-peers-v1','HOME':'/wrong-home'}):
            with self.subTest(environment=environment),patch.dict(os.environ,environment,clear=True),patch.object(peer.subprocess,'Popen') as spawn,patch.object(peer.urllib.request,'build_opener') as http:
                with self.assertRaisesRegex(RuntimeError,'private keyed peer admission required'):
                    peer.exercise({'application_state':True})
                spawn.assert_not_called();http.assert_not_called()

    def test_reopen_uses_same_named_pier_without_boot_only_options(self):
        for number,ship in enumerate(('zod','nec')):
            with self.subTest(ship=ship):
                item={'number':number,'port':18081+number,'pier':Path('/lab/piers/private-'+ship)/ship}
                boot=peer.launch_command(ship,item,False,'http://127.0.0.1:12345')
                reopen=peer.launch_command(ship,item,True,'http://127.0.0.1:12345')
                self.assertEqual(boot[-1],str(item['pier']));self.assertEqual(reopen[-1],boot[-1])
                self.assertEqual(reopen[reopen.index('-p')+1],str(31337+number))
                self.assertEqual(reopen[reopen.index('--http-port')+1],str(18081+number))
                self.assertEqual(boot[boot.index('-w')+1],ship)
                self.assertEqual(boot[boot.index('-H')+1],'galaxies.omabit.invalid')
                self.assertEqual(boot[boot.index('-k')+1],'/lab/secrets/'+str(number)+'.key')
                for flag in ('-H','-w','-k','-e','-B','-c'):
                    self.assertNotIn(flag,reopen)
                for flag in ('-L','--no-conn','--no-dock'):
                    self.assertIn(flag,reopen)

if __name__=='__main__':unittest.main()
