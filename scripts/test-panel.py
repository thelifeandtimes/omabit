#!/usr/bin/env python3
"""Load a closed panel with isolated CLI fixtures on the current Wayland session."""
import json
import datetime
import argparse
import os
import pathlib
import subprocess
import tempfile

ROOT = pathlib.Path(__file__).resolve().parents[1]

FAKE = '''#!/usr/bin/env python3
import datetime,json,os,pathlib,subprocess,sys,time
directory=pathlib.Path(__file__).parent
stamp=os.environ["OMABIT_FIXTURE_TIME"]
names=["0i0","0i9007199254740992","0i9007199254740993","0i18446744073709551615"]
entries=[{"name":name,"digest":str(i+1)*64,"logical_bytes":100+i,"allocated_bytes":4096,"file_count":4 if i==0 else 5,"snapshot_bytes":0 if i==0 else 90,"shared_files":False,"protected":i>=2} for i,name in enumerate(names)]
inventory={"source":"vere-4.6-epoch-v2","pier_id":"a"*32,"digest":"d"*64,"observed_at":stamp,"entries":entries}
comet_inventory=dict(inventory,pier_id="c"*32,digest="e"*64)
if (directory/"inspection.json").exists():
    inventory["observed_at"]=json.loads((directory/"inspection.json").read_text())["finished_at"]
features=["storage-locations","import-pier","delete-pier","bootstrap-selection","host-handoff","runtime-logs","create-fake","fake-ship-options","pier-storage","key-boot","key-boot-options","livenet-import","create-comet","container-cpu","offline-pack","offline-meld","offline-roll","epoch-inventory","epoch-prune-prefix","login-code","import-qualified-fake","maintenance-schedules","comet-compaction","comet-epochs"]
features += ["host-port-suggestion"]
with (directory/"calls.jsonl").open("a") as log:
    log.write(json.dumps(sys.argv[1:])+"\\n")
if "capture" in sys.argv:
    env=dict(os.environ,XDG_RUNTIME_DIR=os.environ["OMABIT_FIXTURE_DESKTOP_RUNTIME"],HYPRLAND_INSTANCE_SIGNATURE=os.environ.get("OMABIT_FIXTURE_HYPRLAND_INSTANCE",os.environ.get("HYPRLAND_INSTANCE_SIGNATURE","")))
    clients=json.loads(subprocess.check_output(["hyprctl","-j","clients"],env=env))
    window=next(c for c in clients if c["pid"]==os.getppid() and c["title"]=="Omabit · Monitor & maintenance")
    active=json.loads(subprocess.check_output(["hyprctl","-j","activeworkspace"],env=env))
    (pathlib.Path(sys.argv[2])/(sys.argv[3]+".json")).write_text(json.dumps({"window":window,"active_workspace":active},indent=2))
    assert window["workspace"]["id"]==active["id"] and window["mapped"] and not window["hidden"], "fixture window must be visible on the active workspace"
    x,y=window["at"]; w,h=window["size"]
    subprocess.run(["grim","-g",f"{x},{y} {w}x{h}",str(pathlib.Path(sys.argv[2])/(sys.argv[3]+".png"))],check=True,env=env,timeout=5)
elif "inventory" in sys.argv:
    while True:
        now=datetime.datetime.now(datetime.timezone.utc).isoformat()
        state="stopped" if (directory/"stopped").exists() else "running"
        inspection=json.loads((directory/"inspection.json").read_text()) if (directory/"inspection.json").exists() else None
        if inspection: inventory["observed_at"]=inspection["finished_at"]
        snapshot={"runner_id":"runner-fixture","mode":"development-managed","instances":[{"id":"alpha","mode":"fake","ship":"~zod","config_revision":2,"applied_revision":2,"desired_state":state,"observed_state":"running" if state=="running" else "stopped","observed_at":now,"health":"healthy","config":{"http_port":18081,"ames_port":34343,"loom":31},"endpoint":"http://127.0.0.1:18081","epochs":inventory,"resources":{"status":"fresh","source":"docker-stats-v1.47","container_id":"c"*64,"observed_at":now,"memory_current_bytes":67108864,"memory_limit_bytes":4294967296,"pids_current":6}}],"operations":[inspection] if inspection else [],"incidents":[]}
        snapshot["instances"][0]["pier_path"]="fakeships/alpha/pier"
        if (directory/"policy.json").exists(): snapshot["instances"][0]["schedule"]=json.loads((directory/"policy.json").read_text())
        snapshot["instances"][0]["resources"].update({"cpu_usage_percent":12.5,"cpu_limit_cores":2})
        snapshot["instances"][0]["runtime_status"]={"initialized":True}
        snapshot["piers"]={"runner_id":"runner-fixture","root":"/var/lib/urbit-runner/piers","observed_at":now,"candidates":[{"path":"fakeships/project/nec/pier","kind":"fake","fingerprint":"f"*64,"busy":False},{"path":"fakeships/project/zod/pier","kind":"fake","fingerprint":"a"*64,"busy":False,"enrolled_by":"alpha"}]}
        snapshot["instances"].append({"id":"comet-fixture","mode":"comet","ship":"~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet","config_revision":2,"applied_revision":2,"desired_state":"stopped","observed_state":"stopped","observed_at":now,"health":"unknown","config":{"http_port":18087,"ames_port":34343,"loom":31},"epochs":comet_inventory})
        if (directory/"owned-registered.json").exists():
            owned=json.loads((directory/"owned-registered.json").read_text())
            started=(directory/"owned-started").exists()
            owned.update({"observed_at":now,"observed_state":"running" if started else "uninitialized","desired_state":"running" if started else "stopped","health":"healthy" if started else "unknown"})
            snapshot["instances"].append(owned)
        for file in directory.glob("created-*.json"):
            created=json.loads(file.read_text());started=(directory/("started-"+created["id"])).exists()
            created.update(observed_at=now,desired_state="running" if started else "stopped",observed_state="running" if started else "uninitialized",health="healthy" if started else "unknown",applied_revision=1 if started else 0)
            snapshot["instances"].append(created)
        snapshot["piers"]["roots"]=[{"id":"default","path":"/var/lib/urbit-runner/piers"}]
        if (directory/"storage-added").exists(): snapshot["piers"]["roots"].append({"id":"ssd","path":"/fixture/storage"})
        if (directory/"deleted-alpha").exists(): snapshot["instances"]=[i for i in snapshot["instances"] if i["id"]!="alpha"]
        available_features=features+(["comet-prune-prefix"] if (directory/"comet-pruning-qualified").exists() else [])
        print(json.dumps({"name":"fixture","runner_id":"runner-fixture","mode":"development-managed","access_role":"operator","features":available_features,"status":"available","checked_at":now,"snapshot":snapshot}),flush=True)
        time.sleep(.2)
elif "suggest-port" in sys.argv:
    time.sleep(.6)
    print(json.dumps({"runner_id":"runner-fixture","http_port":18123,"container_port":8080,"observed_at":stamp}))
elif "storage-root-add" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("storage-root-add")+1]=="ssd"
    assert sys.argv[sys.argv.index("--path")+1]=="/fixture/storage"
    (directory/"storage-added").touch()
    print(json.dumps({"id":"op-storage-root","kind":"storage-root","instance_id":"","request_id":sys.argv[sys.argv.index("--request-id")+1],"status":"succeeded"}))
elif "preview-pier" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("--source")+1]=="/fixture/incoming/zod"
    print(json.dumps({"path":"/fixture/incoming/zod","fingerprint":"b"*64,"bytes":42,"files":3}))
elif "import-pier" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("import-pier")+1]=="import-ui"
    assert sys.argv[sys.argv.index("--source")+1]=="/fixture/incoming/zod"
    assert sys.argv[sys.argv.index("--fingerprint")+1]=="b"*64
    assert sys.argv[sys.argv.index("--path")+1]=="@ssd/fakeships/import-ui/pier"
    assert sys.argv[sys.argv.index("--ship")+1]=="~zod" and "--confirm-stopped" in sys.argv
    print(json.dumps({"id":"op-import-pier","kind":"import-pier","instance_id":"import-ui","request_id":sys.argv[sys.argv.index("--request-id")+1],"status":"queued"}))
elif "delete-pier" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("delete-pier")+1]=="alpha"
    assert sys.argv[sys.argv.index("--revision")+1]=="2" and "--confirm-delete" in sys.argv
    (directory/"deleted-alpha").touch()
    print(json.dumps({"id":"op-delete-pier","kind":"delete-pier","instance_id":"alpha","request_id":sys.argv[sys.argv.index("--request-id")+1],"status":"queued"}))
elif "logs" in sys.argv:
    time.sleep(.6)
    name=sys.argv[sys.argv.index("logs")+1]
    revision=int(sys.argv[sys.argv.index("--revision")+1])
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    lines=["2026-10-02T15:00:00Z boot: "+name,"2026-10-02T15:00:01Z http: fail 504","<plain-text-fixture>"]
    if name=="comet-fixture": lines.insert(0,"found comet: ~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet")
    logs={"runner_id":"runner-fixture","instance_id":name,"config_revision":revision,"source":"docker","observed_at":stamp,"container_id":"c"*64,"kind":"comet-boot","lines":lines,"truncated":False}
    if "--watch" in sys.argv:
        while True:
            print(json.dumps({"runner_id":"runner-fixture","instance_id":name,"config_revision":revision,"logs":logs}),flush=True)
            time.sleep(.2)
    else:
        print(json.dumps(logs))
elif "move-host" in sys.argv:
    assert sys.argv[sys.argv.index("move-host")+1]=="alpha"
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("--to-runner-id")+1]=="runner-move-target"
    assert sys.argv[sys.argv.index("--to-runner")+1]=="remote-move"
    assert sys.argv[sys.argv.index("--revision")+1]=="2" and "--request-id" in sys.argv
    print(json.dumps({"id":"b"*32,"source_runner":"runner-fixture","source_instance":"alpha","destination_runner":"runner-move-target","destination_instance":"alpha","state":"complete","direction":"in"}))
elif "create-comet" in sys.argv:
    assert "--confirm-network" in sys.argv and "--request-id" in sys.argv
    assert sys.argv[sys.argv.index("--expected-runner-id")+1] == "runner-fixture"
    instance=sys.argv[sys.argv.index("create-comet")+1]
    if instance=="taken-on-server":
        assert sys.argv[sys.argv.index("--bootstrap")+1]=="recommended"
        assert sys.argv[sys.argv.index("--pill")+1]=="https://example.invalid/apps.pill"
        print(json.dumps({"error":{"code":"instance_exists","message":"instance ID is already registered"}}),file=sys.stderr)
        sys.exit(1)
    (directory/("created-"+instance+".json")).write_text(json.dumps({"id":instance,"mode":"comet","ship":"~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet","config_revision":1,"config":{"http_port":int(sys.argv[sys.argv.index("--http-port")+1]),"ames_port":34343,"loom":31}}))
    print(json.dumps({"id":"op-comet-fixture","status":"succeeded","instance_id":instance,"request_id":sys.argv[sys.argv.index("--request-id")+1],"kind":"register-comet","config_revision":1}))
elif "create-fake" in sys.argv:
    assert sys.argv[sys.argv.index("--bootstrap")+1]=="recommended"
    assert sys.argv[sys.argv.index("--pill")+1]=="https://example.invalid/apps.pill"
    if sys.argv[sys.argv.index("--ship")+1]=="~foo":
        print(json.dumps({"error":{"code":"client_error","message":"invalid fake Urbit identity"}}),file=sys.stderr)
        sys.exit(2)
    assert sys.argv[sys.argv.index("--ship")+1]=="~bus"
    assert sys.argv[sys.argv.index("--pier-name")+1]=="bus-data"
    assert sys.argv[sys.argv.index("--pier-path")+1]=="fakeships/bus-data/pier"
    instance=sys.argv[sys.argv.index("create-fake")+1]
    (directory/("created-"+instance+".json")).write_text(json.dumps({"id":instance,"mode":"fake","ship":"~bus","config_revision":1,"config":{"http_port":18090,"ames_port":34343,"loom":31}}))
    print(json.dumps({"id":"op-named-fake","status":"succeeded","instance_id":instance,"request_id":sys.argv[sys.argv.index("--request-id")+1],"kind":"register-fake","config_revision":1}))
elif "enroll-pier" in sys.argv:
    assert "--confirm-stopped" in sys.argv and sys.argv[sys.argv.index("--fingerprint")+1]=="f"*64
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("--path")+1]=="fakeships/project/nec/pier"
    print(json.dumps({"id":"op-enroll-pier","instance_id":"nec","request_id":sys.argv[sys.argv.index("--request-id")+1],"kind":"enroll-pier","status":"queued"}))
elif "create-ship" in sys.argv:
    assert "--confirm-network" in sys.argv and sys.argv[sys.argv.index("--key-file")+1]=="/fixture/owned.key"
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert "--http-port" not in sys.argv and "--ames-port" not in sys.argv and sys.argv[sys.argv.index("--pier-path")+1]=="livenet/planets/sampel-palnet/pier"
    assert sys.argv[sys.argv.index("--bootstrap")+1]=="bare" and "--pill" not in sys.argv
    (directory/"owned-registered.json").write_text(json.dumps({"id":"sampel-palnet","ship":"~sampel-palnet","mode":"planet","config_revision":1,"applied_revision":0,"config":{"http_port":18091,"ames_port":34344,"loom":31}}))
    print(json.dumps({"id":"op-key-ship","instance_id":"sampel-palnet","request_id":sys.argv[sys.argv.index("--request-id")+1],"kind":"register-ship","status":"succeeded","config_revision":1}))
elif "operate" in sys.argv and sys.argv[sys.argv.index("operate")+2]=="start":
    instance=sys.argv[sys.argv.index("operate")+1]
    assert instance in ("new-comet","bus-lab","sampel-palnet")
    assert sys.argv[sys.argv.index("--expected-runner-id")+1]=="runner-fixture"
    assert sys.argv[sys.argv.index("--runner")+1]=="fixture"
    assert sys.argv[sys.argv.index("--revision")+1]=="1" and "--request-id" in sys.argv
    if instance=="sampel-palnet": (directory/"owned-started").touch()
    else: (directory/("started-"+instance)).touch()
    print(json.dumps({"id":"op-owned-start","status":"queued","kind":"start","instance_id":instance,"request_id":sys.argv[sys.argv.index("--request-id")+1],"config_revision":1}))
elif "routine-set" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1] == "runner-fixture"
    old=json.loads((directory/"policy.json").read_text()) if (directory/"policy.json").exists() else None
    assert int(sys.argv[sys.argv.index("--revision")+1]) == (old["revision"] if old else 0)
    assert sys.argv[sys.argv.index("--instance-revision")+1] == "2"
    assert "--stop-running=true" in sys.argv and "--resume=false" in sys.argv
    enabled="--enabled=true" in sys.argv
    config={"enabled":enabled,"action":"pack","weekdays":[0],"hour_utc":2,"minute_utc":0,"window_minutes":15,"stop_running":True,"resume":False}
    policy={"id":"policy-fixture","instance_id":"alpha","revision":old["revision"]+1 if old else 1,"instance_revision":2,"config":config}
    if enabled: policy["next_slot"]=(datetime.datetime.now(datetime.timezone.utc)+datetime.timedelta(days=1)).isoformat()
    (directory/"policy.json").write_text(json.dumps(policy))
    print(json.dumps(policy))
elif "inspect-epochs" in sys.argv:
    now=datetime.datetime.now(datetime.timezone.utc).isoformat()
    op={"id":"op-inspection","instance_id":"alpha","kind":"inspect-epochs","status":"succeeded","config_revision":2,"request_id":sys.argv[sys.argv.index("--request-id")+1],"created_at":now,"finished_at":now}
    (directory/"inspection.json").write_text(json.dumps(op))
    print(json.dumps(dict(op,status="queued")))
elif "runners" in sys.argv:
    print(json.dumps([{"name":"fixture","runner_id":"runner-fixture","revision":1,"transport":"ssh","ssh":{"host":"fixture-host"}},{"name":"offline-runner","runner_id":"runner-offline","revision":3,"transport":"ssh","ssh":{"host":"offline-host"}}]))
elif "import-sources" in sys.argv:
    print(json.dumps({"runner_id":"runner-fixture","sources":[{"run_id":"a"*32,"content_digest":"b"*64,"files":7,"bytes":1024,"ship":"~zod","qualified_at":stamp}]}))
elif "import-fake" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1] == "runner-fixture"
    assert sys.argv[sys.argv.index("--source-run")+1] == "a"*32
    assert sys.argv[sys.argv.index("--expected-digest")+1] == "b"*64
    assert "--request-id" in sys.argv and "--confirm-stopped" in sys.argv
    print(json.dumps({"id":"op-import-fixture","status":"queued"}))
elif "copy-code" in sys.argv:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1] == "runner-fixture"
    assert sys.argv[sys.argv.index("--instance")+1] == "alpha"
    assert sys.argv[sys.argv.index("--container-id")+1] == "c"*64
    print(json.dumps({"status":"clipboard_handoff_started","expires_after_seconds":30}),flush=True)
    time.sleep(.05)
elif "ships" in sys.argv:
    print(json.dumps([{"id":"a"*32,"name":"app-fixture","ship":"~zod","origin":"http://127.0.0.1:18085","status":"login_required"}]))
elif "ship-login" in sys.argv:
    assert "--expected-ship-id" in sys.argv
    assert "fixture-only-secret" not in sys.argv
    assert sys.stdin.read() == "fixture-only-secret"
    print(json.dumps({"id":"a"*32,"name":"app-fixture","status":"authenticated"}))
elif "setup-status" in sys.argv:
    print(json.dumps({"execution_model":"owner-user","ready":False,"can_setup":True,"host_present":False,"checks":[]}))
elif "setup" in sys.argv:
    for step in ("check","key","download","authorize","engine","access","services","verify","connect"):
        print(json.dumps({"step":step,"status":"running"}),flush=True)
    print(json.dumps({"step":"ready","status":"complete","result":{"execution_model":"owner-user","ready":True,"can_setup":True,"checks":[],"connection":{"name":"fixture","runner_id":"runner-fixture","mode":"development-managed"},"browser_base":"http://127.0.0.1"}}),flush=True)
elif "theme-status" in sys.argv:
    print(json.dumps({"version":1,"ships":{}}))
elif "theme-preview" in sys.argv:
    print(json.dumps({"id":"omarchy-urbit-theme","name":"Fixture","dark":True,"primary":"#112233","secondary":"#112233","tertiary":"#112233","background":"#112233","surface":"#112233"}))
elif "theme-sync" in sys.argv:
    print(json.dumps({"complete":True,"failed":[]}))
elif "epoch-preview" in sys.argv:
    instance=sys.argv[sys.argv.index("epoch-preview")+1]
    selected_inventory=comet_inventory if instance=="comet-fixture" else inventory
    assert sys.argv[sys.argv.index("--expected-digest")+1] == selected_inventory["digest"]
    cutoff=sys.argv[sys.argv.index("--cutoff")+1]
    index=names.index(cutoff)
    assert index<2
    deletion_supported=instance!="comet-fixture" or (directory/"comet-pruning-qualified").exists()
    if instance=="comet-fixture":
        (directory/"comet-pruning-qualified").touch()
    print(json.dumps({"runner_id":"runner-fixture","instance_id":instance,"config_revision":2,"inventory_digest":selected_inventory["digest"],"observed_at":selected_inventory["observed_at"],"cutoff":cutoff,"selected":entries[:index+1],"retained":entries[index+1:],"selected_logical_bytes":sum(e["logical_bytes"] for e in entries[:index+1]),"selected_allocated_bytes":4096*(index+1),"shared_files":False,"deletion_supported":deletion_supported,"requires_stopped_revalidation":True}))
else:
    assert sys.argv[sys.argv.index("--expected-runner-id")+1] == "runner-fixture"
    assert sys.argv[sys.argv.index("--revision")+1] == "2"
    assert "--request-id" in sys.argv
    if "stop" in sys.argv:
        (directory/"stopped").touch()
    if "epoch-prune" in sys.argv:
        assert "--confirm-history-loss" in sys.argv
        comet=sys.argv[sys.argv.index("epoch-prune")+1]=="comet-fixture"
        assert not comet or (directory/"comet-pruning-qualified").exists()
        assert sys.argv[sys.argv.index("--expected-digest")+1] == (comet_inventory if comet else inventory)["digest"]
        assert sys.argv[sys.argv.index("--cutoff")+1] == names[0 if comet else 1]
    # Simulate an unreadable acknowledgement, with no automatic resubmission.
    print("invalid-acknowledgement" if "meld" in sys.argv else json.dumps({"id":"op-fixture","status":"succeeded"}))
'''


def main():
    compile(FAKE, "fixture-collector", "exec")
    parser = argparse.ArgumentParser()
    parser.add_argument("--windows", action="store_true", help="also briefly map the fixture monitoring window and check all views")
    parser.add_argument("--native", action="store_true", help="exercise the canonical repository entry-point wrappers")
    parser.add_argument("--capture-dir", help="save app-only fixture images while checking mapped windows")
    args = parser.parse_args()
    if args.capture_dir:
        if not args.windows:
            parser.error("--capture-dir requires --windows")
        lock = json.loads(subprocess.check_output(["omarchy", "shell", "lock", "status"], text=True, timeout=5))
        if any(lock.get(key) for key in ("secure", "requested", "locked", "sessionLocked")):
            raise RuntimeError("Unlock the desktop before capturing fixture windows; a locked compositor only captures the lock screen.")
        pathlib.Path(args.capture_dir).mkdir(mode=0o700, parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="omabit-panel-") as temporary:
        directory = pathlib.Path(temporary)
        for name in ("Commons", "Ui", "services"):
            (directory / name).symlink_to(pathlib.Path("/usr/share/omarchy/shell") / name)
        (directory / "Plugin").symlink_to(ROOT / "plugins/omabit")
        helper = directory / "collector-fixture"
        helper.write_text(FAKE)
        helper.chmod(0o700)
        runtime = directory / "runtime"
        runtime.mkdir(mode=0o700)
        # Native Hyprland IPC needs a short socket path within the isolated runtime.
        (runtime / "hypr").mkdir()
        (runtime / "hypr/f").symlink_to(pathlib.Path(os.environ["XDG_RUNTIME_DIR"]) / "hypr" / os.environ["HYPRLAND_INSTANCE_SIGNATURE"])
        picker_files = directory / "picker-fixtures"
        picker_files.mkdir(mode=0o700)
        key_file = picker_files / "owned key λ #1.key"
        pill_file = picker_files / "boot pill #1.pill"
        for path in (key_file, pill_file):
            path.write_text("synthetic picker fixture; never submit or boot\n")
            path.chmod(0o600)
        (directory / "shell.qml").write_text('''import QtQuick
import QtQuick.Dialogs
import Quickshell
import Quickshell.Io
import qs.Commons
import "Plugin" as Plugin
import "Plugin/Model.js" as Model
ShellRoot {
  property bool testWindows: %s
  property string captureDirectory: %s
  property int capturesPending: 0
  property var captures: ({})
  property string capturing: ""
  property url pickerKey: %s
  property url pickerPill: %s
  property var testSettings: ({collectorPath: %s, omabitPath: %s, notifications: false})
  Plugin.Service { id: service; settings: testSettings }
  Plugin.Panel { id: panel; service: service; settings: testSettings; manageIpc: false; bar: null }
  Item { width: 600; visible: false; Plugin.SetupPane { id: setupFixture; width: 600; service: service; mode: "ship" } }
  Component { id: teardownRegistration; Plugin.RegistrationPane { width:600 } }
  readonly property var maintenance: service.monitorWindow.maintenancePane
  property int phase: -2
  onPhaseChanged: console.log("Fixture phase", phase)
  property string uncertainRequest: ""
  function capture(name) {
    if (!captureDirectory || captures[name]) return false
    capturesPending=1; capturing=name
    captureProcess.command=[testSettings.collectorPath,"capture",captureDirectory,name]; captureProcess.running=true
    return true
  }
  Process {
    id: captureProcess
    stderr: SplitParser { onRead: function(line) { console.warn("Capture: "+line) } }
    onExited: function(code,status) {
      if (code !== 0 || status !== 0) { Qt.exit(1); throw new Error("fixture capture failed") }
      captures[capturing]=true; capturesPending=0
    }
  }
  function control(object, name, visited) {
    visited = visited || []
    if (!object || visited.indexOf(object) >= 0) return null
    visited.push(object)
    if (object.objectName === name) return object
    for (var key of ["children", "data", "contentItem"]) {
      var children = object[key]
      if (!children) continue
      if (typeof children.length !== "number") children = [children]
      for (var i = 0; i < children.length; i++) { var found = control(children[i], name, visited); if (found) return found }
    }
    return null
  }
  Timer {
    interval: 300; repeat: true; running: true
     onTriggered: {
       if (capturesPending) return
      if (phase === -2 && service.localSetupStatus && service.connections.length === 1) {
        if (service.setupCheckRequested || service.machineUse!=="") throw new Error("fresh onboarding already chose machine/check state")
        service.setMachineUse("laptop")
        if (!service.setUpLocal() || service.setUpLocal()) throw new Error("local setup dispatch/busy guard failed")
        phase = -1
      } else if (phase === -1 && !service.settingUp) {
        if (service.localSetupError || !service.localSetupStatus.ready || service.localSetupStep !== "ready") throw new Error("local setup did not finish with verified readiness")
        if (service.runnerPreferences["runner-fixture"].browserBase !== "http://127.0.0.1") throw new Error("local browser default was not applied")
        console.log("PASS: guided local setup streams progress, prevents duplicate dispatch and applies verified local connection defaults")
        phase = 0
      } else if (phase === 0 && service.connections.length === 1) {
        if (!service.registrations.length) return
        var teardown=teardownRegistration.createObject(null,{service:service,connection:service.connections[0]})
        teardown.kind="fake";teardown.destroy()
        if (!service.saveRunnerPresentation(service.connections[0], "Fixture", "Local fixture", "http://127.0.0.1")) throw new Error("presentation settings failed")
        if (Model.shipURL(service.connections[0],service.connections[0].snapshot.instances[0],service.registrations,service.runnerPreferences) !== "http://127.0.0.1:18081") throw new Error("browser host composition failed")
        if (service.connections[0].snapshot.instances[0].health !== "healthy") throw new Error("bad fixture state")
         var codeButton=control(service.monitorWindow,"monitor-copy-code")
         if (!codeButton || !codeButton.enabled) throw new Error("full-window +code action unavailable")
         var fleetCode=control(panel,"fleet-copy-code-alpha")
         if (!fleetCode || !fleetCode.enabled || fleetCode.hint.indexOf("Update")>=0 || fleetCode.connection.runner_id!==codeButton.connection.runner_id) throw new Error("menubar +code differs from full-panel selection")
         codeButton.clicked()
         if (service.codeSubject !== "[~zod] (alpha)") throw new Error("code handoff not ship-bound")
        service.operate(service.connections[0], service.connections[0].snapshot.instances[0], "stop")
        phase = 1
       } else if (phase === 1 && service.lastResult.indexOf("op-fixture") >= 0) {
         if (service.toastText !== "Copied to sensitive clipboard for 30s") throw new Error("clipboard toast missing or verbose")
        var c = service.connections[0], i = c.snapshot.instances[0]
        if (i.observed_state !== "stopped") return
         service.monitorWindow.page = "maintenance"
         var inspect = control(maintenance,"inspect-epochs")
         if (!inspect || !inspect.enabled) throw new Error("inspection action missing")
         inspect.clicked(); phase = 101
       } else if (phase === 101 && service.epochInspection && service.epochInspection.status === "ready") {
         if (!maintenance.retentionTarget || !control(maintenance,"history-review").visible || control(maintenance,"inspection-feedback").text.indexOf("Inspection complete")<0) throw new Error("inspection did not open its result")
         console.log("PASS: asynchronous inspection opens a bound inventory review with clear completion feedback")
         var c=service.connections[0], i=c.snapshot.instances[0]
         maintenance.reviewMaintenance(c, i, "pack")
        if (service.maintain(Object.assign({}, maintenance.maintenanceTarget, {revision: 999}))) throw new Error("stale maintenance submitted")
        var run = control(maintenance, "confirm-maintenance")
        if (!run || !run.enabled) throw new Error("maintenance confirmation missing")
        run.clicked()
        phase = 2
      } else if (phase === 2 && !service.busy && service.lastResult.indexOf("pack:") === 0) {
        service.maintain(Model.maintenanceTarget(service.connections[0], service.connections[0].snapshot.instances[0], "meld", service.now))
        uncertainRequest = service.requestIdentity
        phase = 3
      } else if (phase === 3 && !service.busy && service.lastError) {
        if (!uncertainRequest || service.requestIdentity !== uncertainRequest) throw new Error("uncertain request ID lost")
        maintenance.reviewRetention(service.connections[0], service.connections[0].snapshot.instances[0])
        maintenance.selectCutoff("0i9007199254740992")
        if (!service.previewEpochs(maintenance.retentionTarget)) throw new Error("preview rejected")
        phase = 4
      } else if (phase === 4 && service.canConfirmPrune) {
        if (service.prunePreview(false)) throw new Error("prune without consent")
        var replacement = Object.assign({}, service.connections[0], {runner_id: "runner-replacement"})
        service.connections = [replacement]
        if (service.epochPreview || service.prunePreview(true)) throw new Error("preview survived identity change")
        phase = 5
      } else if (phase === 5 && service.connections[0].runner_id === "runner-fixture") {
        maintenance.reviewRetention(service.connections[0], service.connections[0].snapshot.instances[0])
        maintenance.selectCutoff("0i0")
        service.previewEpochs(maintenance.retentionTarget)
        phase = 6
      } else if (phase === 6 && service.canConfirmPrune) {
        maintenance.selectCutoff("0i9007199254740992")
        if (service.epochPreview || service.prunePreview(true)) throw new Error("preview survived cutoff change")
        service.previewEpochs(maintenance.retentionTarget)
        phase = 7
      } else if (phase === 7 && service.canConfirmPrune) {
        var consent = control(maintenance, "confirm-history-loss"), remove = control(maintenance, "delete-prefix")
        if (!consent || consent.checked || !remove || remove.enabled) throw new Error("prune consent not explicit")
        consent.checked = true
        if (!remove.enabled) throw new Error("confirmed deletion disabled")
        remove.clicked()
        if (service.epochPreview || maintenance.retentionTarget) throw new Error("submitted review retained")
        phase = 8
      } else if (phase === 8 && !service.busy && service.lastResult.indexOf("epoch-prune:") === 0) {
        service.monitorWindow.page="setup";service.monitorWindow.setupPane.mode="apps"
        phase = 9
      } else if (phase === 9) {
        if (!control(service.monitorWindow.setupPane,"applications-coming-soon").visible || control(service.monitorWindow.setupPane,"shipLoginCode")) throw new Error("Application Connections is not a Coming Soon stub")
        console.log("PASS: real QML maintenance confirmation, immutable pruning review, error/request retention, and Coming Soon application stub")
        if (!service.loadImportSources(service.connections[0])) throw new Error("import catalogue request failed")
        phase = 20
      } else if (phase === 20 && service.importCatalogue && !service.busy) {
        var imported = Model.importTarget(service.importCatalogue,service.importCatalogue.sources[0],"imported",18086,service.now)
        if (service.importFake(imported,false)) throw new Error("import without consent")
        if (service.canImport(Object.assign({},imported,{runner:"replaced"}))) throw new Error("import identity changed")
        if (!service.importFake(imported,true)) throw new Error("confirmed import rejected")
        phase = 21
      } else if (phase === 21 && !service.busy && service.lastResult.indexOf("import-fake:") === 0) {
        if (service.importCatalogue) throw new Error("submitted import catalogue retained")
        console.log("PASS: import catalogue identity, bound source and explicit confirmation")
        if (service.registerComet(service.connections[0],"new-comet",18088,false)) throw new Error("comet without acknowledgement")
        if (service.registerComet(service.connections[0],"comet-fixture",18088,true)) throw new Error("duplicate instance dispatched")
        if (service.registerComet(service.connections[0],"new-comet",18087,true)) throw new Error("duplicate port dispatched")
        if (!service.registerComet(service.connections[0],"new-comet",18088,true)) throw new Error("comet registration failed")
        phase = 22
      } else if (phase === 22 && !service.busy && !service.pendingFirstBoot && !service.bootTarget && service.lastResult.indexOf("first-boot:") === 0) {
        console.log("PASS: explicit comet boot acknowledgement registers then starts exactly once")
        service.monitorWindow.selectedKey="runner-fixture/alpha"
        service.monitorWindow.page = "maintenance"
        maintenance.section = "routine"
         maintenance.routinePane.edit()
         if (control(maintenance.routinePane,"routine-enabled")) throw new Error("routine still asks for enable-after-save checkbox")
        var save = control(maintenance,"save-routine")
        if (!save || !save.enabled) throw new Error("routine editor unavailable")
        save.clicked(); phase = 23
       } else if (phase === 23 && !service.busy && service.lastResult.indexOf("routine-set:") === 0) {
         if (!maintenance.routinePane.policy) return
         if (service.routineOutcome.status !== "saved" || control(maintenance.routinePane,"routine-policy-status").text !== "Enabled" || !maintenance.routinePane.policy.config.stop_running) throw new Error("saved policy was not enabled with implied stop permission")
         control(maintenance.routinePane,"toggle-routine").clicked(); phase=231
       } else if (phase === 231 && !service.busy && !maintenance.routinePane.policy.config.enabled) {
         if (control(maintenance.routinePane,"routine-policy-status").text !== "Saved · disabled") throw new Error("explicit disable not reflected")
         control(maintenance.routinePane,"toggle-routine").clicked(); phase=232
       } else if (phase === 232 && !service.busy && maintenance.routinePane.policy.config.enabled) {
         if (maintenance.routinePane.policy.revision !== 3) throw new Error("policy toggle lost revision binding")
         if (control(maintenance.routinePane,"routine-next-run").text.indexOf("Next run:") !== 0) throw new Error("enabled policy timing not visible")
         console.log("PASS: saved policy card shows timing/status and enables/disables with explicit revision-bound actions")
         console.log("PASS: saving enables routine maintenance and implies stop permission while retaining the resume choice")
        maintenance.section = "manual"
        var c = service.connections[0], comet = c.snapshot.instances.filter(function(i) { return i.id === "comet-fixture" })[0]
        service.monitorWindow.selectedKey = c.runner_id + "/" + comet.id
        maintenance.reviewMaintenance(c,comet,"roll")
        var roll = control(maintenance,"confirm-maintenance")
        if (!roll || !roll.enabled) throw new Error("comet roll confirmation unavailable")
        roll.clicked(); phase = 24
      } else if (phase === 24 && !service.busy && service.lastResult.indexOf("roll:") === 0) {
        var c = service.connections[0], comet = c.snapshot.instances.filter(function(i) { return i.id === "comet-fixture" })[0]
        maintenance.reviewRetention(c,comet); maintenance.selectCutoff("0i0")
        if (!service.previewEpochs(maintenance.retentionTarget)) throw new Error("comet preview refused")
        phase = 25
      } else if (phase === 25 && !service.busy && service.epochPreview) {
        var consent = control(maintenance,"confirm-history-loss"), remove = control(maintenance,"delete-prefix")
        if (service.epochPreview.deletion_supported || service.canConfirmPrune || !consent || consent.enabled || !remove || remove.enabled || service.prunePreview(true)) throw new Error("comet preview enabled deletion")
        console.log("PASS: comet roll confirmation and preview render with deletion consent/action disabled")
        phase = 26
      } else if (phase === 26 && Model.hasFeature(service.connections[0],"comet-prune-prefix")) {
        if (service.canConfirmPrune) throw new Error("new capability silently upgraded the old preview")
        var c = service.connections[0], comet = c.snapshot.instances.filter(function(i) { return i.id === "comet-fixture" })[0]
        maintenance.reviewRetention(c,comet); maintenance.selectCutoff("0i0")
        if (!service.previewEpochs(maintenance.retentionTarget)) throw new Error("qualified comet preview refused")
        phase = 27
      } else if (phase === 27 && !service.busy && service.canConfirmPrune) {
        var consent = control(maintenance,"confirm-history-loss"), remove = control(maintenance,"delete-prefix")
        if (!consent || consent.checked || !consent.enabled || !remove || remove.enabled || service.prunePreview(false)) throw new Error("comet pruning skipped explicit consent")
        consent.checked = true
        if (!remove.enabled) throw new Error("confirmed comet deletion remained disabled")
        remove.clicked(); phase = 28
      } else if (phase === 28 && !service.busy && service.lastResult.indexOf("epoch-prune:") === 0) {
        if (service.epochPreview || maintenance.retentionTarget) throw new Error("submitted comet review retained")
        console.log("PASS: new comet pruning capability requires fresh preview and explicit history-loss confirmation")
        maintenance.clearReview(); service.monitorWindow.selectedKey = "runner-fixture/alpha"
        setupFixture.mode = "ship"; setupFixture.shipKind = "comet"
        var name = control(setupFixture,"registration-name"), port = control(setupFixture,"registration-port")
         name.text = "taken-on-server"; port.text = "18089"
         service.settings=Object.assign({},service.settings,{recommendedPillUrl:"https://example.invalid/apps.pill"})
         var choice=control(control(setupFixture,"registration-form"),"bootstrap-choice")
         choice.currentIndex=1;choice.activated(1);phase=29
       } else if (phase === 29) {
        var consent = control(setupFixture,"comet-consent")
        consent.checked = true
        if (!consent.text || consent.indicator.x !== consent.leftPadding || consent.contentItem.x + consent.contentItem.leftPadding <= consent.indicator.x + consent.indicator.width) throw new Error("consent indicator overlaps its label")
        var submit = control(setupFixture,"register-instance")
        if (!submit.enabled) throw new Error("registration form unavailable")
        submit.clicked(); phase = 30
      } else if (phase === 30 && !service.busy) {
        if (setupFixture.mode !== "ship" || !service.registrationOutcome || service.registrationOutcome.status !== "failed" || control(setupFixture,"registration-name").text !== "taken-on-server" || control(setupFixture,"registration-port").text !== "18089") throw new Error("registration rejection lost the form or input")
        console.log("PASS: consent is left-aligned and a backend registration conflict retains the form and values")
        setupFixture.shipKind = "fake"
        control(setupFixture,"registration-name").text = "bus-lab"
        control(setupFixture,"registration-port").text = "18090"
         control(setupFixture,"registration-fake-ship").text = "~foo"
         var location=control(control(setupFixture,"registration-form"),"storage-location-selector")
         location.currentIndex=location.count-1;location.activated(location.currentIndex)
         control(setupFixture,"registration-pier").text = "bus-data"
        phase = 31
       } else if (phase === 31) {
         control(setupFixture,"comet-consent").checked=true
        var submit = control(setupFixture,"register-instance")
        if (!submit.enabled) throw new Error("named fake form unavailable")
        submit.clicked(); phase = 32
       } else if (phase === 32 && !service.busy) {
         if (setupFixture.mode !== "ship" || service.registrationOutcome.status !== "failed" || !control(setupFixture,"register-instance").enabled) throw new Error("local validation became an uncertain registration")
         console.log("PASS: local fake-name validation retains a correctable form without claiming an uncertain mutation")
         control(setupFixture,"registration-fake-ship").text = "~bus"
         phase = 33
       } else if (phase === 33) {
         var submit = control(setupFixture,"register-instance")
         if (!submit.enabled) throw new Error("corrected named fake form unavailable")
         submit.clicked(); phase = 34
       } else if (phase === 34 && !service.busy && !service.pendingFirstBoot && !service.bootTarget) {
         if (setupFixture.mode !== "ship" || service.registrationOutcome.status !== "succeeded" || service.registrationOutcome.ship !== "~bus" || service.registrationOutcome.pier !== "fakeships/bus-data/pier") throw new Error("named fake result lost identity or path")
        console.log("PASS: fake identity, instance name and pier directory are submitted independently")
         if (testWindows) { service.monitorWindow.selectedKey="runner-fixture/alpha";service.openMonitor(null,null,"overview"); phase = 10 } else Qt.quit()
        } else if (phase === 10) {
          if (!service.monitorWindow.visible) throw new Error("monitor failed to map")
          var runtime=control(service.monitorWindow,"runtime-pane")
          if (!runtime.logs) return
          if (runtime.logs.instance_id!=="alpha" || control(runtime,"runtime-log-text").text.indexOf("boot: alpha")<0 || control(runtime,"runtime-log-text").textFormat!==TextEdit.PlainText) throw new Error("runtime logs missing or not plain text")
          control(runtime,"runtime-logs-follow").clicked()
          if (runtime.following) throw new Error("logs cannot be paused")
          control(runtime,"runtime-logs-follow").clicked()
          console.log("PASS: per-ship runtime logs load as bounded plain text and support pause/follow")
         var nav=control(service.monitorWindow,"nav-ship-alpha")
         if (!nav || nav.text !== "[~zod]" || !nav.font.italic || control(service.monitorWindow,"container-memory").value !== "N/A") throw new Error("fleet identity or stopped RAM rendering failed")
         if (control(service.monitorWindow,"monitor-refresh").text !== "↻" || control(service.monitorWindow,"monitor-ship-context").instanceName !== "alpha") throw new Error("header/context refinement missing")
         service.monitorWindow.selectedKey="runner-fixture/comet-fixture"; phase=14
        } else if (phase === 14) {
          var runtime=control(service.monitorWindow,"runtime-pane")
          if (!runtime.logs) return
           if (runtime.logs.instance_id!=="comet-fixture" || control(runtime,"runtime-log-text").text.indexOf("boot: alpha")>=0) throw new Error("log tail crossed a ship selection")
           if (runtime.reportedComet!=="~doznec-salbud-salwex-sogref--tasweb-navfun-batwet-sigfet") throw new Error("Public comet identity was not extracted from Vere output")
         var title=control(service.monitorWindow,"monitor-ship-name")
         if (title.paintedWidth>title.width+1 || title.text !== service.connections[0].snapshot.instances[1].ship) throw new Error("comet title overflow")
         if (capture("comet-overview")) return
          service.monitorWindow.width=Style.space(780); phase=15
       } else if (phase === 15) {
         var title=control(service.monitorWindow,"monitor-ship-name")
         if (title.paintedWidth>title.width+1) throw new Error("narrow comet title overflow")
         if (capture("comet-narrow")) return
         control(service.monitorWindow,"monitor-setup").clicked(); phase=16
       } else if (phase === 16) {
         if (service.monitorWindow.page !== "setup" || !service.monitorWindow.setupPane.active || !service.monitorWindow.setupPane.visible) throw new Error("full-window setup unavailable")
         if (control(service.monitorWindow,"fleet-marker-comet-fixture").text !== "") throw new Error("setup retained ship selection marker")
         if (capture("setup")) return
         service.monitorWindow.selectedKey="runner-fixture/alpha"
         service.monitorWindow.page = "maintenance"; maintenance.section = "routine"
         Qt.callLater(function(){service.monitorWindow.revealItem(maintenance.routinePane)})
         phase = 11
       } else if (phase === 11) {
         if (capture("routine")) return
         service.monitorWindow.page = "activity"; phase = 12
       } else if (phase === 12) {
         var subject=control(service.monitorWindow.activityPane,"activity-subject")
         if (!subject || subject.text !== "[~zod]") throw new Error("activity subject is not the ship identity")
         var eventTime=control(service.monitorWindow.activityPane,"activity-time"), eventAction=control(service.monitorWindow.activityPane,"activity-action"), eventDetails=control(service.monitorWindow.activityPane,"activity-details"), eventContext=control(service.monitorWindow.activityPane,"activity-context")
         if (Math.abs(eventTime.x+eventTime.width-eventTime.parent.width)>1 || eventTime.y>1) throw new Error("activity timestamp is not top-right")
         if (eventDetails.mapToItem(eventAction.parent,0,0).y<eventAction.y+eventAction.height || Math.abs(eventContext.x+eventContext.width-eventContext.parent.width)>1) throw new Error("activity details/context positioning incorrect")
         if (control(service.monitorWindow,"fleet-marker-alpha").text !== "" || control(service.monitorWindow,"nav-ship-alpha").font.bold) throw new Error("activity retained ship selection styling")
         if (control(service.monitorWindow.activityPane,"activity-context").status !== "succeeded") throw new Error("activity context lost status")
         if (capture("activity")) return
        control(service.monitorWindow,"activity-runner-filter").popup.open(); phase = 13
      } else if (phase === 13) {
        var filter = control(service.monitorWindow,"activity-runner-filter")
        filter.currentIndex = 1; filter.activated(1); filter.popup.close()
        if (service.monitorWindow.activityPane.runnerFilter !== "runner-fixture") throw new Error("runner activity filter failed")
         console.log("PASS: monitoring window maps and renders overview, maintenance, activity and themed filter popup")
         console.log("PASS: full-window setup, primary @p/sidebar instance labels, stopped N/A readings, +code routing and wrapped comet titles")
         service.monitorWindow.page="maintenance"; maintenance.section="manual"
         maintenance.reviewRetention(service.connections[0],service.connections[0].snapshot.instances[0]); phase=17
       } else if (phase === 17) {
         var scroll=control(service.monitorWindow,"monitor-scroll")
         scroll.contentY=Math.max(0,scroll.contentHeight-scroll.height); phase=18
       } else if (phase === 18) {
         if (capture("history")) return
         service.monitorWindow.page="overview"
         var hint=control(service.monitorWindow,"omabit-tooltip")
         hint.text="Running · healthy. Omabit tooltips follow the active Omarchy palette."
         hint.open(); phase=181
       } else if (phase === 181) {
         var hint=control(service.monitorWindow,"omabit-tooltip")
         if (!hint.visible) return
         if (String(hint.background.color) !== String(Plugin.Design.raised)) throw new Error("tooltip does not use Omabit theme")
         if (capture("tooltip")) return
         hint.close()
         service.monitorWindow.revealItem(control(service.monitorWindow,"cpu-chart-card")); phase=182
       } else if (phase === 182) {
         var chart=control(service.monitorWindow,"cpu-chart")
         if (control(chart,"chart-y-0").text.indexOf("%%")<0 || !/^[0-9]{2}:[0-9]{2}:[0-9]{2}$/.test(control(chart,"chart-x-2").text)) throw new Error("CPU axes missing percentage or time units")
         if (capture("cpu-chart")) return
         service.monitorWindow.revealItem(control(service.monitorWindow,"memory-chart-card")); phase=183
       } else if (phase === 183) {
         var chart=control(service.monitorWindow,"memory-chart")
         if (control(chart,"chart-y-0").text.indexOf("MiB")<0 || control(chart,"chart-x-2").text === "—") throw new Error("memory axes missing units or timestamps")
         if (capture("memory-chart")) return
         console.log("PASS: CPU and memory charts have labeled units/time axes; activity/setup clear ship selection markers")
         phase=19
       } else if (phase === 19 && capturesPending === 0) {
         service.monitorWindow.page="setup";service.monitorWindow.setupPane.mode="storage"
         phase=300
       } else if (phase === 300) {
         var storage=control(service.monitorWindow.setupPane,"pier-storage")
          var already=control(storage,"review-pier-fakeships/project/zod/pier")
         if (already.visible || already.enabled) throw new Error("already-enrolled pier offered enrollment when moved flag was absent")
          control(storage,"review-pier-fakeships/project/nec/pier").clicked()
          var review=control(storage,"pier-enrollment-fakeships/project/nec/pier")
         if (!review.visible || control(review,"enroll-pier-submit").enabled) throw new Error("inline review missing or auto-authorized")
         service.monitorWindow.revealItem(review);phase=3001
       } else if (phase === 3001) {
         if (capture("inline-enrollment")) return
          var storage=control(service.monitorWindow.setupPane,"pier-storage"),review=control(storage,"pier-enrollment-fakeships/project/nec/pier")
         var consent=control(review,"enroll-pier-consent");consent.checked=true;consent.toggled()
         if (!control(review,"enroll-pier-submit").enabled) throw new Error("confirmed enrollment remained disabled")
         control(review,"enroll-pier-submit").clicked();phase=301
       } else if (phase === 301 && !service.busy) {
         var storage=control(service.monitorWindow.setupPane,"pier-storage")
          if (!storage.submitted || service.lastResult.indexOf("enroll-pier:")!==0 || control(control(storage,"pier-enrollment-fakeships/project/nec/pier"),"enroll-pier-submit").enabled) throw new Error("enrollment selection or non-retry guard lost")
          if (capture("pier-storage")) return
          console.log("PASS: discovered pier enrollment requires fresh fingerprint/runner binding and explicit stopped-source confirmation")
          service.monitorWindow.setupPane.mode="ship";service.monitorWindow.setupPane.shipKind="key"
          var form=control(service.monitorWindow.setupPane,"owned-key-form")
           control(form,"key-identity").text="~sampel-palnet";form.keyFile="/fixture/owned.key"
           if(control(form,"key-register").enabled)throw new Error("key boot skipped consent")
           control(form,"key-boot-consent").checked=true
          if (control(form,"key-start").enabled || !control(form,"key-register").enabled) throw new Error("Start enabled before registration or registration unavailable")
          control(form,"key-register").clicked()
          if (control(form,"key-start").enabled) throw new Error("Start enabled while registration was submitting")
          phase=302
        } else if (phase === 302 && !service.busy && !service.pendingFirstBoot && !service.bootTarget) {
          if (service.registrationOutcome.kind!=="owned" || service.registrationOutcome.status!=="succeeded") throw new Error("key registration acknowledgement not bound")
          var form=control(service.monitorWindow.setupPane,"owned-key-form")
          if (!form.registeredInstance) return
          if (form.registeredInstance.observed_state!=="running") return
          if (control(form,"key-start").enabled || control(form,"key-register").enabled || service.monitorWindow.page!=="overview") throw new Error("owned registration did not automatically boot/open management")
          service.monitorWindow.page="setup"
          control(form,"key-register-another").clicked()
          if (control(form,"key-start").enabled) throw new Error("Start remained enabled on a blank registration form")
          service.lastError="old key-flow error"
         service.monitorWindow.setupPane.mode="home";phase=3021
       } else if (phase === 3021) {
         var setup=service.monitorWindow.setupPane
         if (service.lastError || control(setup,"setup-card-key") || control(setup,"setup-card-ship")) throw new Error("setup retained stale errors or duplicate provisioning cards")
         control(setup,"setup-add-ship").clicked()
         if (setup.mode!=="ship") throw new Error("Add a ship did not open provisioning")
         control(setup,"add-ship-method-comet").clicked()
         if (!control(setup,"registration-form").visible) throw new Error("comet flow is not in Add a ship")
         control(setup,"add-ship-method-key").clicked();phase=303
       } else if (phase === 303) {
         if (!control(service.monitorWindow.setupPane,"owned-key-form").visible) throw new Error("key flow is not in Add a ship")
         if (control(service.monitorWindow.setupPane,"key-advanced").visible) throw new Error("advanced boot choices were not collapsed")
         if (capture("key-boot")) return
          control(service.monitorWindow.setupPane,"key-identity").text="~dibpen-fogweg-dozzod-sarlev"
           var keyForm=control(service.monitorWindow.setupPane,"owned-key-form")
           if (keyForm.instanceName!=="dibpen-fogweg-dozzod-sarlev" || keyForm.destination!=="livenet/moons/dibpen-fogweg-dozzod-sarlev/pier") throw new Error("generated moon defaults incorrect")
           if (control(keyForm,"key-start").enabled) throw new Error("another registered identity enabled Start for an unregistered moon")
          keyForm.keyFile="/previous/key.key"
          var picker=control(keyForm,"key-file-picker")
          if (!(picker.options & FileDialog.DontUseNativeDialog)) throw new Error("key Browse would enter the crashing native GTK/GVFS path")
          picker.currentFolder=Qt.resolvedUrl("picker-fixtures/")
          control(keyForm,"key-file-browse").clicked();phase=3031
        } else if (phase === 3031) {
          var keyForm=control(service.monitorWindow.setupPane,"owned-key-form"),picker=control(keyForm,"key-file-picker")
          if (!picker.visible) throw new Error("key Browse did not open a dialog")
          picker.selectedFile=pickerKey;picker.reject();phase=3032
        } else if (phase === 3032) {
          var keyForm=control(service.monitorWindow.setupPane,"owned-key-form"),picker=control(keyForm,"key-file-picker")
          if (picker.visible || keyForm.keyFile!=="/previous/key.key") throw new Error("canceling key Browse changed the form")
          control(keyForm,"key-file-browse").clicked();phase=3033
        } else if (phase === 3033) {
          var picker=control(service.monitorWindow.setupPane,"key-file-picker")
          if (!picker.visible) throw new Error("key Browse did not reopen")
          picker.selectedFile=pickerKey;picker.accept();phase=3034
        } else if (phase === 3034) {
          var keyForm=control(service.monitorWindow.setupPane,"owned-key-form")
          if (control(keyForm,"key-file-picker").visible || keyForm.keyFile!==Model.localPath(pickerKey) || control(keyForm,"key-file-location").text!==keyForm.keyFile) throw new Error("key selection lost the decoded local path")
          control(service.monitorWindow.setupPane,"key-advanced-toggle").clicked()
          service.monitorWindow.revealItem(control(service.monitorWindow.setupPane,"key-advanced"));phase=304
       } else if (phase === 304) {
         if (!control(service.monitorWindow.setupPane,"key-advanced").visible) throw new Error("advanced configuration did not expand")
          if (capture("key-advanced")) return
          console.log("PASS: key boot routes only a key-file path through QML and checks the registration acknowledgement")
           service.settings=Object.assign({},service.settings,{recommendedPillUrl:"",defaultBootPill:""})
           var choice=control(control(service.monitorWindow.setupPane,"owned-key-form"),"bootstrap-choice")
           if (choice.model[0]!=="bare bootstrap" || choice.model[1]!=="Recommended Apps") throw new Error("wrong pill choices")
           choice.currentIndex=1;choice.activated(1);phase=3041
         } else if (phase === 3041) {
           var form=control(service.monitorWindow.setupPane,"owned-key-form")
           if (form.selectedPill!=="" || form.problem.indexOf("not configured")<0 || control(form,"key-register").enabled) throw new Error("missing recommendation silently fell back")
           service.settings=Object.assign({},service.settings,{recommendedPillUrl:"https://example.invalid/apps.pill"});phase=3042
         } else if (phase === 3042) {
           var form=control(service.monitorWindow.setupPane,"owned-key-form")
           if (form.selectedPill!=="https://example.invalid/apps.pill" || form.problem!=="") throw new Error("configured Recommended Apps selection unavailable")
           var choice=control(form,"bootstrap-choice");choice.currentIndex=0;choice.activated(0);phase=3043
         } else if (phase === 3043) {
           var form=control(service.monitorWindow.setupPane,"owned-key-form")
           if (form.selectedPill!=="" || form.problem!=="") throw new Error("bare bootstrap inherited configured recommendation")
           phase=3044
         } else if (phase === 3044) {
           console.log("PASS: non-native key Browse preserves paths; bootstrap selector requires a configured recommendation and keeps bare independent")
          var form=control(service.monitorWindow.setupPane,"owned-key-form")
          form.keyFile="";control(form,"key-identity").text="~sampel-palnet"
          if (service.registrationOutcome || !form.registered || control(form,"key-start").enabled) throw new Error("restored running identity offered a duplicate Start")
          var selected=form.connection
          form.connection={name:"offline-runner",runner_id:"runner-offline"}
          if (control(form,"key-start").enabled) throw new Error("Start crossed the selected runner identity")
          form.connection=selected
          form.advanced=false
          service.monitorWindow.revealItem(control(form,"key-start"));phase=3045
        } else if (phase === 3045) {
          if (capture("key-registered")) return
          var start=control(service.monitorWindow.setupPane,"key-start")
          if (start.enabled) throw new Error("running ship offered another Start")
          start.clicked()
          if (start.enabled) throw new Error("Start stayed enabled during dispatch")
          start.clicked();phase=3046
        } else if (phase === 3046 && !service.busy) {
          var form=control(service.monitorWindow.setupPane,"owned-key-form")
          if (form.registeredInstance.observed_state!=="running") return
          if (control(form,"key-start").enabled) throw new Error("running ship offered a duplicate Start")
          console.log("PASS: confirmed owned creation boots once, opens progress, and restored forms cannot duplicate Start")
          service.monitorWindow.page="setup";service.monitorWindow.setupPane.mode="manage";phase=305
       } else if (phase === 305) {
         var setup=service.monitorWindow.setupPane
         if (!control(setup,"managed-runner-offline-runner") || control(setup,"ship-http-port")) throw new Error("Manage runners lost local enrollments or still contains ship controls")
         if (!control(service.monitorWindow,"edit-ship-ports")) throw new Error("ship configuration control was lost")
         if (capture("manage-runners")) return
         control(setup,"add-runner-connection").clicked()
         if (setup.mode!=="runner") throw new Error("Manage runners cannot add a connection")
          console.log("PASS: one Add a ship entry exposes comet/key provisioning; Manage runners includes offline collector enrollments without ship controls")
          var source=service.connections[0]
          service.connections=service.connections.concat([{name:"remote-move",runner_id:"runner-move-target",mode:"development-managed",status:"available",access_role:"operator",checked_at:new Date().toISOString(),features:["host-handoff"],snapshot:{runner_id:"runner-move-target",mode:"development-managed",instances:[],operations:[],incidents:[]}}])
          service.monitorWindow.selectedKey="runner-fixture/alpha";service.monitorWindow.page="overview";phase=306
        } else if (phase === 306) {
          var pane=control(service.monitorWindow,"host-move-pane")
          if (!control(pane,"review-host-move").enabled) throw new Error("stopped source host move unavailable")
          control(pane,"review-host-move").clicked()
          var chooser=control(pane,"host-move-destination");chooser.currentIndex=1;chooser.activated(1)
          phase=307
        } else if (phase === 307) {
          var pane=control(service.monitorWindow,"host-move-pane"), submit=control(pane,"submit-host-move")
          if (!submit.enabled || pane.selectedDestination.runner!=="runner-move-target") throw new Error("host move selection was not pinned: "+pane.problem)
          submit.clicked();submit.clicked();phase=308
        } else if (phase === 308 && !service.busy) {
          if (!service.hostMoveOutcome || service.hostMoveOutcome.status!=="complete" || service.lastError) throw new Error("host move receipt failed")
           console.log("PASS: host move review pins both runners and source revision, dispatches once, and reports the stopped destination")
           service.monitorWindow.page="setup";service.monitorWindow.setupPane.mode="storage";phase=309
        } else if (phase === 309) {
           var storage=control(service.monitorWindow,"pier-storage")
           control(storage,"storage-root-name").text="ssd";control(storage,"storage-root-path").text="/fixture/storage"
           control(storage,"storage-root-add").clicked();phase=310
        } else if (phase === 310 && !service.busy) {
           var storage=control(service.monitorWindow,"pier-storage")
           if (!storage.catalogue.roots || storage.catalogue.roots.length!==2) return
           control(storage,"open-pier-import").clicked()
           var form=control(storage,"external-pier-import")
           control(form,"import-pier-source").text="/fixture/incoming/zod"
           control(form,"import-pier-review").clicked();phase=311
        } else if (phase === 311 && !service.busy) {
           var form=control(service.monitorWindow,"external-pier-import")
           if (!form.current) throw new Error("external source review was not bound to this runner/path")
           control(form,"import-pier-name").text="import-ui";control(form,"import-pier-ship").text="~zod"
           control(form,"import-pier-destination").text="fakeships/import-ui/pier";control(form,"import-pier-port").text="18094"
           var roots=control(form,"storage-location-selector");roots.currentIndex=1;roots.activated(1)
           control(form,"import-pier-consent").checked=true
           if (!control(form,"import-pier-submit").enabled) throw new Error("reviewed external import cannot be submitted")
           control(form,"import-pier-submit").clicked();control(form,"import-pier-submit").clicked();phase=312
        } else if (phase === 312 && !service.busy) {
           if (service.lastError || service.storageOutcome.kind!=="import-pier" || service.storageOutcome.status!=="accepted") throw new Error("external import acknowledgement failed: "+service.lastError+" "+JSON.stringify(service.storageOutcome))
           console.log("PASS: storage location selection and reviewed host-path import dispatch once with content/runner/destination binding")
           service.monitorWindow.page="overview";service.monitorWindow.selectedKey="runner-fixture/alpha";phase=313
        } else if (phase === 313) {
           var pane=control(service.monitorWindow,"overview-delete-pier")
           control(pane,"delete-pier-review").clicked()
           control(pane,"delete-pier-name").text="wrong-name"
           if (control(pane,"delete-pier-submit").enabled) throw new Error("deletion did not require exact instance name")
           control(pane,"delete-pier-name").text="alpha"
           if (!control(pane,"delete-pier-submit").enabled) throw new Error("stopped reviewed deletion is disabled")
           control(pane,"delete-pier-submit").clicked();control(pane,"delete-pier-submit").clicked();phase=314
        } else if (phase === 314 && !service.busy) {
           if (service.lastError || service.storageOutcome.kind!=="delete-pier" || service.storageOutcome.status!=="accepted") throw new Error("deletion acknowledgement failed")
           console.log("PASS: destructive deletion requires stopped instance and typed identity, and submits once with runner/revision binding")
           service.monitorWindow.visible=false;Qt.quit()
      }
    }
  }
}''' % ("true" if args.windows else "false",json.dumps(str(pathlib.Path(args.capture_dir).absolute()) if args.capture_dir else ""),json.dumps(key_file.as_uri()),json.dumps(pill_file.as_uri()),json.dumps(str(helper)),json.dumps(str(helper))))
        if args.native:
            qml = (directory / "shell.qml").read_text()
            qml = qml.replace('import "Plugin" as Plugin', 'import "Plugin" as Plugin\nimport ' + json.dumps((ROOT / "native").as_uri()) + ' as Native')
            qml = qml.replace("Plugin.Service {", "Native.Service {").replace("Plugin.Panel {", "Native.Panel {")
            qml = qml.replace("if (capturesPending) return", "if (capturesPending) return\n        if (String(service.toolsDirectory)!==" + json.dumps((ROOT / "native/bin").as_uri()) + ") throw new Error(\"native tool directory mismatch\")")
            (directory / "shell.qml").write_text(qml)
        # Quickshell PanelWindow has no offscreen backend. Connect to the current
        # compositor but keep the panel closed; use a separate IPC/log runtime.
        display = os.environ.get("WAYLAND_DISPLAY", "")
        if not display:
            raise RuntimeError("a Wayland session is required for the closed-panel load test")
        if not display.startswith("/"):
            display = str(pathlib.Path(os.environ["XDG_RUNTIME_DIR"]) / display)
        env = dict(os.environ, QT_QPA_PLATFORM="wayland", WAYLAND_DISPLAY=display, XDG_RUNTIME_DIR=str(runtime), HYPRLAND_INSTANCE_SIGNATURE="f", OMABIT_FIXTURE_HYPRLAND_INSTANCE=os.environ["HYPRLAND_INSTANCE_SIGNATURE"], NO_COLOR="1",
                   XDG_STATE_HOME=str(directory / "state"), OMABIT_FIXTURE_DESKTOP_RUNTIME=os.environ["XDG_RUNTIME_DIR"], OMABIT_FIXTURE_TIME=datetime.datetime.now(datetime.timezone.utc).isoformat())
        try:
            result = subprocess.run(["qs", "--no-color", "-p", str(directory)], env=env, capture_output=True, text=True, timeout=60)
        except subprocess.TimeoutExpired as error:
            raise RuntimeError((error.stdout or b"").decode(errors="replace") + (error.stderr or b"").decode(errors="replace")) from None
        text = result.stdout + result.stderr
        if result.returncode != 0 or "PASS: real QML" not in text:
            raise RuntimeError(text)
        for line in text.splitlines():
            if "PASS:" in line:
                print(line)
        if any(message in text for message in ("Error:", "Unable to assign", "is not a type", "Cannot assign to", "Binding loop detected")):
            raise RuntimeError(text)
        calls = [json.loads(line) for line in (directory / "calls.jsonl").read_text().splitlines()]
        mutations = [c for c in calls if "operate" in c or "epoch-prune" in c or "import-fake" in c or "create-comet" in c or "create-fake" in c or "enroll-pier" in c or "create-ship" in c or "move-host" in c or "storage-root-add" in c or "import-pier" in c or "delete-pier" in c]
        expected_mutations=21 if args.windows else 14
        assert len(mutations) == expected_mutations, mutations
        assert [c[c.index("operate") + 2] for c in mutations if "operate" in c] == ["stop", "inspect-epochs", "pack", "meld", "start", "roll", "start"] + (["start"] if args.windows else [])
        assert len({c[c.index("--request-id") + 1] for c in mutations}) == expected_mutations
        assert len([c for c in calls if "routine-set" in c]) == 3
        assert all("fixture-only-secret" not in c for c in calls)
        assert not any(any(a in c for a in ("ships","ship-login","theme-sync")) for c in calls), "Coming Soon UI started application-session work"
        print("PASS: fixture CLI audit confirms bound maintenance and registration, local duplicate refusal, no automatic retry, and no secret in argv")


if __name__ == "__main__":
    main()
