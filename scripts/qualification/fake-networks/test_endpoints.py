"""Offline rejection checks for the exact publication contract."""
import copy
import unittest
from endpoints_case import published_bindings


def sample():
    ports={'8081/tcp':[{'HostIp':'127.0.0.1','HostPort':'18281'}],'8082/tcp':[{'HostIp':'127.0.0.1','HostPort':'18282'}]}
    return {'HostConfig':{'PortBindings':copy.deepcopy(ports)},'NetworkSettings':{'Ports':dict(copy.deepcopy(ports),**{'80/tcp':None})}}

class EndpointTests(unittest.TestCase):
    def test_exact_loopback_and_unpublished_exposed_port(self):
        published_bindings(sample(),{8081:18281,8082:18282})

    def test_wildcard_ipv6_wrong_port_and_extra_mapping_refused(self):
        for scope,field in [('HostConfig','PortBindings'),('NetworkSettings','Ports')]:
            for fault in ('wildcard','ipv6','wrong-port','extra'):
                with self.subTest(scope=scope,fault=fault):
                    value=sample();ports=value[scope][field]
                    if fault=='wildcard':ports['8081/tcp'][0]['HostIp']='0.0.0.0'
                    elif fault=='ipv6':ports['8081/tcp'].append({'HostIp':'::','HostPort':'18281'})
                    elif fault=='wrong-port':ports['8081/tcp'][0]['HostPort']='18282'
                    else:ports['12321/tcp']=[{'HostIp':'127.0.0.1','HostPort':'18285'}]
                    with self.assertRaisesRegex(RuntimeError,'publication'):
                        published_bindings(value,{8081:18281,8082:18282})

    def test_missing_publication_is_not_success(self):
        value=sample();value['NetworkSettings']['Ports']['8081/tcp']=None
        with self.assertRaisesRegex(RuntimeError,'publication'):
            published_bindings(value,{8081:18281,8082:18282})

if __name__=='__main__':unittest.main()
