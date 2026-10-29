"""Keep one real DM Bridge alive across a separate assisted approval."""
import json
from pathlib import Path
import sys
import threading

from qualify import check_namespace


def main():
    check_namespace()
    root=Path('/lab/ui-lab/home/.config/omarchy/plugins/urbit.dms.ui-lab')
    sys.path.insert(0,str(root))
    from bridge import Bridge
    from session_store import SessionStore
    from instance import INSTANCE
    assert INSTANCE.profile=='ui-lab'
    bridge=Bridge(session_store=SessionStore())  # Inert native launcher adapter.
    events=[]
    bridge.emit=lambda event,**data: events.append((event,data))
    subscribed=threading.Event()
    original=bridge.on_fact
    def on_fact(outer,event_id):
        if isinstance(outer,dict) and outer.get('response')=='subscribe' and not outer.get('err'):
            subscribed.set()
        original(outer,event_id)
    bridge.on_fact=on_fact
    try:
        bridge.command({'action':'restore'})
        assert bridge.ship=='' and not events,'bridge did not begin empty'
        print(json.dumps({'stage':'empty','ok':True}),flush=True)
        assert json.loads(sys.stdin.readline(4097))=={'action':'refresh'}
        bridge.command({'action':'refresh'})
        assert bridge.ship=='~zod','running bridge did not adopt the approved account'
        assert any(event=='login' and data.get('restored') for event,data in events),'missing restored login event'
        assert any(event=='refreshed' for event,data in events),'missing direct refreshed event'
        assert subscribed.wait(7),'running bridge subscription not confirmed'
        assert isinstance(bridge.scry('chat','/dm'),list),'running bridge direct read failed'
        print(json.dumps({'stage':'restored','ok':True}),flush=True)
        assert json.loads(sys.stdin.readline(4097))=={'action':'stop'}
    finally:
        bridge.reset_runtime()
        bridge.pool.shutdown(wait=True,cancel_futures=True)
    print(json.dumps({'stage':'stopped','ok':True}),flush=True)


if __name__=='__main__':
    try:main()
    except Exception:
        print(json.dumps({'ok':False,'error':'running-bridge-failed'}),flush=True)
        sys.exit(1)
