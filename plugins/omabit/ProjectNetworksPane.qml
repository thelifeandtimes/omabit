import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property bool active: false
  signal addShip()
  readonly property alias controller: reader
  readonly property var runners: service ? service.connections : []
  readonly property var selectedRunner: runner.currentIndex >= 0 ? runners[runner.currentIndex] || null : null
  readonly property var selectedProject: project.currentIndex >= 0 ? reader.projects[project.currentIndex] || null : null
  onSelectedProjectChanged: reader.clearDetail()
  property var requested:null
  function present(connection,id) {
    if(!connection||!Model.validId(id))return
    requested={runner:connection.runner_id,name:connection.name,project:id}
    runner.currentIndex=runners.findIndex(function(c){return c.runner_id===connection.runner_id&&c.name===connection.name})
    Qt.callLater(function(){if(root.requested&&reader.canList&&!reader.busy)reader.refresh()})
  }
  function selectRequested() {
    if(!requested||!selectedRunner||selectedRunner.runner_id!==requested.runner||selectedRunner.name!==requested.name)return
    var index=reader.projects.findIndex(function(p){return p.id===root.requested.project})
    if(index<0)return
    project.currentIndex=index;requested=null
    Qt.callLater(function(){if(root.selectedProject&&reader.canReadStatus&&!reader.busy)reader.inspect(root.selectedProject)})
  }
  Connections {target:reader;function onProjectsChanged(){root.selectRequested()}}
  spacing: Style.space(12)
  function actionTitle(kind) {
    var names = {"register-project":"Reserve project", "prepare-project":"Prepare network", "start-project-anchor":"Start network anchor", "prepare-project-member-storage":"Prepare ship storage", "launch-project-member":"Start ship", "observe-project-member":"Check ship readiness", "stop-project-member":"Stop ship", "observe-project-member-stop-exit":"Check ship exit", "retire-project-member":"Retire stopped container", "resume-project-member":"Resume ship", "observe-resumed-project-member":"Check resumed ship", "observe-current-project-member":"Check current ship", "recover-project-anchor":"Recover network", "observe-project-recovery":"Check recovery", "continue-project-recovery":"Continue recovery", "recover-project-member":"Recover ship"}
    return names[kind] || "Saved operation"
  }
  function savedState(status) { return {queued:"Queued",running:"In progress",succeeded:"Completed",failed:"Failed",interrupted:"Interrupted"}[status] || status }
  function timestamp(value) { return Model.shortTime(value) }
  readonly property var creationReceipt: reader.detail ? reader.detail.operations.filter(function(o){return o.kind==="register-project"&&Model.validId(o.request_id)})[0] || null : null
  readonly property bool creationIncomplete: !!creationReceipt && reader.detail.operations.filter(function(o){return o.kind==="observe-project-member"&&o.status==="succeeded"}).length < reader.detail.project.members.length && !reader.detail.operations.some(function(o){return ["failed","interrupted"].indexOf(o.status)>=0})

  ProjectNetworks { id: reader; service: root.service; connection: root.selectedRunner; active: root.active && root.visible }

  Label { text: "Project networks"; font.pixelSize: Style.space(19); font.bold: true }
  Label { width: parent.width; secondary: true; text: "Development ships that share a private project network." }
  Label { width: parent.width; visible: !!root.service && !!root.service.profileName && root.service.profileName !== "default"; text: root.service ? "Profile: " + (root.service.profileName || "default") : ""; caption: true; secondary: true }
  SelectField { id: runner; objectName: "projects-runner"; width: parent.width; model: root.runners; textRole: "name"; enabled: root.active }
  Label {
    width: parent.width; secondary: true
    text: !root.selectedRunner ? "Connect a runner to view its projects." : !reader.available ? "This runner is unavailable. Saved project history cannot be refreshed." : !Model.hasFeature(root.selectedRunner, "project-reservations-v1") ? "This runner does not support project inventory. Update it before using this view." : ""
    visible: text !== ""
  }
  readonly property bool duplicateLabel: reader.projects.some(function(p){return p.label.toLowerCase()===newProjectName.text.trim().toLowerCase()})
  Label {width:parent.width;visible:root.duplicateLabel;text:"This project already exists. Choose it below.";color:Design.warning}
  Field { id:newProjectName; objectName:"new-project-name"; width:parent.width;placeholderText:"New project name";enabled:!!root.service&&!root.service.busy }
  Flow { width:parent.width;spacing:Style.space(8)
    ActionButton { objectName:"create-empty-project";text:"Create project network";enabled:!!root.selectedRunner&&!!root.service&&!root.service.busy&&!root.duplicateLabel&&Model.validId(newProjectName.text.trim());onClicked:root.service.projectCreation.start(root.selectedRunner,newProjectName.text.trim(),[],true,"") }
    ActionButton { text:"Add a fake ship →";quiet:true;onClicked:root.addShip() }
  }
  Label { width:parent.width;text:"Each project has its own Docker network. Its fake ships share administrative loopback access.";caption:true;secondary:true }
  ProjectCreationStatus { width: parent.width; service: root.service }
  Connections {target:root.service?root.service.projectCreation:null;function onFinished(){if(reader.canList&&!reader.busy)reader.refresh()}}
  ActionButton { objectName: "projects-refresh"; text: reader.busy ? "Reading…" : "Refresh projects"; enabled: reader.canList && !reader.busy; onClicked: reader.refresh() }
  Label { objectName: "projects-error"; width: parent.width; text: reader.error; color: Design.danger; visible: text !== "" }
  Label { width: parent.width; visible: reader.readAt !== "" && reader.projects.length === 0; text: "No saved projects on this runner."; secondary: true }
  Label { objectName: "projects-history-boundary"; width: parent.width; visible: reader.readAt !== ""; text: "Saved reservations and operation history. Live ship health has not been checked."; secondary: true }
  SelectField { id: project; objectName: "projects-selection"; width: parent.width; model: reader.projects.map(function(p){return {label:p.label+(reader.projects.filter(function(other){return other.label.toLowerCase()===p.label.toLowerCase()}).length>1?" · "+p.id.slice(-8):"")}}); textRole: "label"; visible: reader.projects.length > 0; enabled: !reader.busy }
  Label { width: parent.width; visible: !!root.selectedProject; text: root.selectedProject ? root.selectedProject.members.concat(root.selectedProject.additional_members||[]).map(function(m) { return m.ship + " · " + m.instance_id + " · HTTP " + m.http_port }).join("\n") : ""; secondary: true }
  Flow {
    width:parent.width;spacing:Style.space(8)
    Repeater {model:root.selectedProject?root.selectedProject.members.concat(root.selectedProject.additional_members||[]):[];ActionButton {required property var modelData;text:"Open "+modelData.ship;enabled:!!root.selectedRunner&&!!root.service&&root.service.observing;onClicked:{var url=Model.shipURL(root.selectedRunner,{endpoint:"http://127.0.0.1:"+modelData.http_port},root.service.registrations,root.service.runnerPreferences);if(url)Qt.openUrlExternally(url)}}}
  }
  ActionButton { objectName: "projects-history"; text: "Read saved history"; visible: !!root.selectedProject; enabled: reader.canReadStatus && !reader.busy; onClicked: reader.inspect(root.selectedProject) }
  Label { width: parent.width; visible: !!root.selectedProject && reader.canList && !reader.canReadStatus; text: "This runner can list projects but needs project-history support to show their operations."; secondary: true }
  Column {
    width: parent.width; spacing: Style.space(8); visible: !!reader.detail
    ActionButton {
      visible:root.creationIncomplete
      text:"Continue creating this project"
      enabled:!!root.service&&!!root.service.projectCreation&&!root.service.busy&&!!root.selectedRunner&&Model.writable(root.selectedRunner,root.service.now)
      onClicked:root.service.projectCreation.start(root.selectedRunner,reader.detail.project.label,reader.detail.project.members,true,root.creationReceipt.request_id)
    }
    Label { objectName: "projects-history-title"; text: reader.detail ? "Saved history · " + reader.detail.project.label : ""; font.bold: true }
    Label { width: parent.width; caption: true; secondary: true; text: reader.detail ? "Read " + root.timestamp(reader.detail.read_at) : "" }
    Repeater {
      model: reader.detail ? reader.detail.operations : []
      delegate: Column {
        required property var modelData
        width: root.width; spacing: Style.space(3)
        Label { width: parent.width; text: root.actionTitle(modelData.kind) + (modelData.instance_id ? " · " + modelData.instance_id : ""); font.bold: true }
        Label { width: parent.width; text: "Saved: " + root.savedState(modelData.status) + " · " + root.timestamp(modelData.finished_at || modelData.created_at); secondary: true }
      }
    }
    Label { objectName: "projects-older"; width: parent.width; visible: !!reader.detail && reader.detail.has_older_operations; text: "Showing the newest 128 operations. Older receipts remain on the runner."; secondary: true }
  }
}
