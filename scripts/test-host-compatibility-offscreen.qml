import QtQuick
import QtQuick.Controls
import "Plugin" as Plugin
import "Plugin/Model.js" as Model

ApplicationWindow {
  id: test
  visible: true; width: 900; height: 900
  property int phase: 0
  property var fixtures: (@@HOST_FIXTURES@@)
  property var report: fixtures.pending.host_compatibility.report
  property var ship: fixtures.pending.snapshot.instances[0]
  property var connection: Model.observation(JSON.stringify(fixtures.pending))
  function control(object,name,visited) {
    visited=visited || []
    if(!object || visited.indexOf(object)>=0)return null
    visited.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue
      if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,visited);if(found)return found}
    }
    return null
  }
  Item {
    id: root
    property var settings: ({})
    property double now: Date.parse(test.fixtures.pending.checked_at)
    property bool busy: false
    property bool activationEnabled: true
    property bool localSetupAllowed: true
    property bool observing: true
    property bool settingUp: false
    readonly property bool checkingSetup: localStatus.running
    property bool setupCheckRequested: false
    property var localSetupStatus: ({execution_model:"owner-user",ready:false,can_setup:true,host_present:false,checks:[],compatibility:test.report})
    property string localSetupStep: ""
    property string localSetupMessage: ""
    property string localSetupError: ""
    property string localStatusOutput: ""
    property string machineUse: "always-on"
    property string collector: "fixture-collector"
    property var registrations: [{name:"fixture",runner_id:"runner-fixture"}]
    property bool registrationsLoaded: true
    property var connections: []
    property string feedError: ""
    property bool refreshPending: false
    property bool refreshing: false
    property var importCatalogue: null
    property var pendingImportConnection: null
    property int registrationChecks: 0
    QtObject { id: localStatus; property bool running: false; property var command: [] }
    QtObject { id: feed; property bool running: false; property var command: [] }
    function themeCommand(args) { return ["fixture-omabit","--format","json"].concat(args) }
    function command(args) { return ["fixture-collector","--format","json"].concat(args) }
    function clearEpochPreview() {}
    function loadRegistrations() { registrationChecks++ }
    function setMachineUse(value) { machineUse=value }
    @@SERVICE_METHODS@@
  }
  Column {
    width: 800
    Plugin.HostSetupPane { id: setup; width: parent.width; service: root }
    Plugin.RuntimePane { id: runtime; width: parent.width; service: root; instance: test.ship; connection: test.connection; active:false }
  }
  Timer {
    interval: 50; running: true; repeat: true
    onTriggered: {
      if(phase===0) {
        if(setup.detailsOpen || control(setup,"setup-install-approval").checked || control(setup,"local-setup-primary").enabled)throw new Error("Existing setup approval/layout changed")
        if(control(setup,"local-host-compatibility").visible || control(setup,"setup-progress-checklist").visible)throw new Error("Fresh setup was replaced by technical diagnostics or claimed progress")
        control(setup,"setup-view-details").clicked()
        if(!setup.detailsOpen || control(setup,"host-compatibility-facts").text.indexOf("provider: deb")<0 || control(setup,"host-compatibility-facts").text.indexOf("5:29.9.1-1")<0)throw new Error("Structured local details missing")
        if(control(setup,"host-compatibility-guidance").text!=="Review Debian native package evidence.")throw new Error("Distribution guidance replaced")
        var details=control(runtime,"runtime-status-detail").text
        if(details.indexOf("Recovery")>=0 || control(runtime,"host-compatibility-summary").text.indexOf("restart pending")<0)throw new Error("Host update mistaken for pier recovery")
        if(Model.canOperate(connection,ship,"stop",root.now)!==connection.host_compatibility.report.can_stop || Model.canOperate(connection,Object.assign({},ship,{observed_state:"stopped"}),"start",root.now)!==connection.host_compatibility.report.can_mutate)throw new Error("Backend Stop/new-work permissions were replaced by UI rules")
        control(setup,"setup-install-approval").checked=true
        if(!control(setup,"local-setup-primary").enabled)throw new Error("Metadata runtime permissions replaced core setup prerequisites")
        root.settingUp=true;root.localSetupStep="authorize"
        if(!control(setup,"setup-progress-checklist").visible || control(setup,"setup-progress-state-0").text!=="✓" || control(setup,"setup-progress-state-2").text!=="✓" || control(setup,"setup-progress-state-3").text!=="4")throw new Error("Authorization phase lost the checked-off setup steps")
        root.settingUp=false
        console.log("PASS: real setup/runtime rows show native facts separately from recovery and retain controls")
        root.localSetupStatus=Object.assign({},root.localSetupStatus,{ready:true,can_setup:false,connection:{name:"fixture",runner_id:"runner-fixture"}})
        test.connection=Model.observation(JSON.stringify(fixtures.older))
        phase=1
      } else if(phase===1) {
        if(control(setup,"local-setup-primary").text!=="Add a ship →")throw new Error("Package drift lost connected setup")
        if(!control(setup,"setup-progress-checklist").visible || control(setup,"setup-progress-state-5").text!=="✓")throw new Error("Completed setup lost its completed checklist")
        if(control(runtime,"host-compatibility-summary").text.indexOf("Update this runner")<0 || Model.instances(connection).length!==1)throw new Error("Older runner lost inventory or upgrade hint")
        root.refresh()
        if(!localStatus.running || localStatus.command.join(" ")!=="fixture-omabit --format json setup-status" || root.setupCheckRequested)throw new Error("Refresh did not use read-only existing helper")
        if(root.registrationChecks!==1)throw new Error("Existing refresh flow lost")
        test.connection=Model.observation(JSON.stringify(fixtures.metadata))
        phase=2
      } else {
        if(!feed.running || feed.command.join(" ")!=="fixture-collector --format json inventory --watch --interval 2s")throw new Error("Default collector polling requires an opt-in or stopped refreshing")
        if(control(runtime,"host-compatibility-summary").text.indexOf("permissions are unresolved")<0 || !Model.canOperate(connection,ship,"stop",root.now) || !Model.canOperate(connection,Object.assign({},ship,{observed_state:"stopped"}),"start",root.now))throw new Error("Metadata-only report disabled existing controls")
        console.log("PASS: older runners stay observable and existing refresh rechecks local metadata")
        console.log("PASS: default collector polling revalidates hosts; unresolved metadata remains diagnostic")
        Qt.quit()
      }
    }
  }
}
