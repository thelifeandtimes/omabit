import copy
from types import SimpleNamespace
import unittest
from runner_app_state import PHASES, begin_phase, execute_phase, restore_recovery, save_recovery

class RunnerAppPhases(unittest.TestCase):
    def test_four_phases_commit_before_work_and_keep_prior_records(self):
        state={'format':1,'attempts':[]};disk={};effects=[]
        def commit(value):disk.clear();disk.update(value)
        for phase in PHASES:
            old=copy.deepcopy(state)
            def work(current):
                self.assertEqual(disk['attempts'][-1],{'phase':phase,'status':'running'})
                self.assertFalse(disk['ok']);effects.append(phase)
                current['proofs']=current.get('proofs',[])+[phase]
            result=execute_phase(state,phase,commit,work)
            self.assertEqual(state,old);state=result
            self.assertEqual(disk,state)
            self.assertEqual(state['attempts'][:-1],old['attempts'])
        self.assertTrue(state['ok']);self.assertEqual(effects,list(PHASES))
    def test_interruption_never_replays_a_mutation(self):
        for fault in ('first-commit','effect','last-commit'):
            with self.subTest(fault=fault):
                disk={'format':1,'attempts':[]};effects=[];commits=0
                def commit(value):
                    nonlocal disk,commits
                    commits+=1
                    if (fault=='first-commit' and commits==1) or (fault=='last-commit' and commits==2):raise OSError('synthetic interrupted write')
                    disk=copy.deepcopy(value)
                def work(current):
                    effects.append('write')
                    if fault=='effect':raise RuntimeError('lost acknowledgement')
                with self.assertRaises((OSError,RuntimeError)):execute_phase(disk,PHASES[0],commit,work)
                self.assertEqual(len(effects),0 if fault=='first-commit' else 1)
                if fault!='first-commit':
                    self.assertEqual(disk['attempts'][-1]['status'],'running')
                    for phase in PHASES:
                        with self.assertRaises(ValueError):begin_phase(disk,phase)
    def test_reordered_replayed_and_foreign_phase_history_is_refused(self):
        for state,phase in [({'format':True,'attempts':[]},PHASES[0]),({'format':1,'attempts':[]},PHASES[1]),({'format':1,'attempts':[{'phase':PHASES[0],'status':'complete'}]},PHASES[0]),({'format':1,'attempts':[{'phase':'foreign','status':'complete'}]},PHASES[1])]:
            with self.subTest(state=state),self.assertRaises(ValueError):begin_phase(state,phase)
    def test_work_cannot_rewrite_earlier_phase_binding(self):
        state={'format':1,'attempts':[]};saved=[]
        def work(current):current['attempts'][0]['phase']='forged'
        with self.assertRaises(ValueError):execute_phase(state,PHASES[0],saved.append,work)
        self.assertEqual(len(saved),1);self.assertEqual(saved[0]['attempts'][0]['phase'],PHASES[0])
    def test_restoration_keeps_original_cursor_and_acknowledged_entries(self):
        entry='omabit-state-'+'a'*24
        original=SimpleNamespace(entries=[entry],channel='/~/channel/omabit-state-'+'b'*32,cursor=7,report={'writes':[{'entry':entry,'receipt':'omabit-receipt-'+'c'*32,'state':'acknowledged'}]})
        state={'format':1,'attempts':[{'phase':PHASES[0],'status':'complete'}]};save_recovery(state,original)
        restored=SimpleNamespace();restore_recovery(state,restored)
        self.assertEqual(vars(original),vars(restored));restored.entries.append('changed');self.assertEqual(state['recovery']['entries'],[entry])
        for mutate in [lambda s:s['entries'].append({}),lambda s:s['report']['writes'][0].update(state='pending'),lambda s:s.update(cursor=True),lambda s:s.update(channel='/~/channel/unrelated'),lambda s:s['report']['writes'][0].update(entry='omabit-state-'+'d'*24)]:
            broken=copy.deepcopy(state);mutate(broken['recovery'])
            with self.assertRaises(ValueError):restore_recovery(broken,SimpleNamespace())

class PeerPhaseRecovery(unittest.TestCase):
    def test_peer_cycle_binding_is_immutable_and_uncertain_work_cannot_repeat(self):
        from runner_app_state import PEER_PHASES
        state={'format':1,'attempts':[],'peer_cycle':True}
        for phase in PHASES:state=execute_phase(state,phase,lambda v:None,lambda v:None)
        self.assertFalse(state['ok'])
        saved=[]
        def change_mode(current):current['peer_cycle']=False
        with self.assertRaises(ValueError):execute_phase(state,'before-peer',saved.append,change_mode)
        self.assertEqual(len(saved),1)
        for phase in PEER_PHASES:
            with self.assertRaises(ValueError):begin_phase(saved[0],phase)
        state=execute_phase(state,'before-peer',lambda v:None,lambda v:None)
        self.assertFalse(state['ok'])
        state=execute_phase(state,'after-peer',lambda v:None,lambda v:None)
        self.assertTrue(state['ok'])
        with self.assertRaises(ValueError):begin_phase(state,'after-peer')


class AnchorPhaseRecovery(unittest.TestCase):
    def test_four_cycles_preserve_phase_binding_and_never_repeat_uncertain_write(self):
        from runner_app_state import ANCHOR_PHASES,PEER_PHASES
        state={'format':1,'attempts':[],'peer_cycle':True,'anchor_cycle':True}
        for phase in PEER_PHASES:state=execute_phase(state,phase,lambda v:None,lambda v:None)
        self.assertFalse(state['ok'])
        saved=[]
        def lose_reply(current):raise RuntimeError('lost fourth-cycle write reply')
        with self.assertRaises(RuntimeError):execute_phase(state,'before-anchor',saved.append,lose_reply)
        for phase in ANCHOR_PHASES:
            with self.assertRaises(ValueError):begin_phase(saved[0],phase)
        def change_mode(current):current['anchor_cycle']=False
        with self.assertRaises(ValueError):execute_phase(state,'before-anchor',lambda v:None,change_mode)
        state=execute_phase(state,'before-anchor',lambda v:None,lambda v:None)
        state=execute_phase(state,'after-anchor',lambda v:None,lambda v:None)
        self.assertTrue(state['ok'])

if __name__=='__main__':unittest.main()
