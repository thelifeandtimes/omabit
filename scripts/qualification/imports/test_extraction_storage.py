import unittest
from unittest.mock import patch
from extract_results import admit_conversion

class ConversionStorage(unittest.TestCase):
    def test_budget_includes_whole_virtual_disk_and_verification_reserve(self):
        for gib in (2, 8, 12):
            with self.subTest(gib=gib), patch('storage.admit', return_value={'available_bytes':20*1024**3}) as admit:
                result=admit_conversion('/tmp',{'format':'qcow2','virtual-size':gib*1024**3})
                admit.assert_called_once_with('/tmp',(gib+2)*1024**3)
                self.assertEqual(result['conversion_virtual_bytes'],gib*1024**3)
    def test_old_fixed_budget_would_admit_exhaustion_but_geometry_refuses(self):
        with patch('storage.available', return_value={'available_bytes':5125001216}):
            with self.assertRaisesRegex(RuntimeError,'insufficient'):
                admit_conversion('/tmp',{'format':'qcow2','virtual-size':12*1024**3})
    def test_invalid_geometry_refused_before_storage_or_conversion(self):
        for size in (None,True,0,-512,1,'12884901888',65*1024**3):
            with self.subTest(size=size),patch('storage.admit') as admit:
                with self.assertRaises(ValueError):admit_conversion('/tmp',{'format':'qcow2','virtual-size':size})
                admit.assert_not_called()
        with self.assertRaises(ValueError):admit_conversion('/tmp',{'format':'raw','virtual-size':12*1024**3})

if __name__=='__main__':unittest.main()
