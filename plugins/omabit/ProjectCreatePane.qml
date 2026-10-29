import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id:root
  property var service
  property var connection
  readonly property var controller:service?service.projectCreation:null
  property bool expanded:false
  spacing:Style.space(10)
  readonly property var members: {
    var names=identities.text.trim().split(/[\s,]+/),port=Number(firstPort.text),prefix=projectName.text.trim()
    return names.map(function(ship,index){return {instance_id:prefix+"-"+ship.replace(/^~/,""),ship:ship,http_port:port+index}})
  }
  readonly property string problem: {
    if(!Model.validId(projectName.text.trim())||projectName.text.trim().length>45)return "Use a short project name with letters, numbers, hyphens or underscores."
    if(members.length<2||members.length>16)return "Choose 2–16 distinct fake galaxies."
    var seen={};for(var i=0;i<members.length;i++){var m=members[i];if(!/^~[a-z]{3}$/.test(m.ship)||seen[m.ship])return "Enter distinct fake galaxies, such as ~zod, ~nec.";seen[m.ship]=true;if(!isFinite(m.http_port)||m.http_port%1!==0||m.http_port<1024||m.http_port>65535)return "Choose an available port range between 1024 and 65535."}
    return ""
  }
  ActionButton {text:root.expanded?"New project −":"Create a connected fake project +";primary:!root.expanded;enabled:!root.controller||!root.controller.busy;onClicked:root.expanded=!root.expanded}
  Column {
    width:parent.width;spacing:Style.space(8);visible:root.expanded
    Label {width:parent.width;secondary:true;text:"Choose the fresh ships together. They can message one another inside this private project; another project can reuse the same fake identities."}
    Label {text:"Project name";caption:true}
    Field {id:projectName;objectName:"project-create-name";width:parent.width;placeholderText:"demo";enabled:!!root.controller&&!root.service.busy;onTextEdited:consent.checked=false}
    Label {text:"Fake identities";caption:true}
    Field {id:identities;objectName:"project-create-ships";width:parent.width;text:"~zod, ~nec";enabled:!!root.controller&&!root.service.busy;onTextEdited:consent.checked=false}
    Label {text:"First host HTTP port";caption:true}
    Field {id:firstPort;objectName:"project-create-port";width:parent.width;text:"18381";validator:IntValidator {bottom:1024;top:65535} enabled:!!root.controller&&!root.service.busy;onTextEdited:consent.checked=false}
    Label {width:parent.width;caption:true;secondary:true;text:root.problem||root.members.map(function(m){return m.ship+" · "+m.instance_id+" · HTTP "+m.http_port}).join("\n")}
    CheckField {id:consent;width:parent.width;enabled:!!root.controller&&!root.service.busy;text:"Download the pinned Vere runtime and core bootstrap if needed, then create and start these ships. Project members trust each other and share administrative loopback access."}
    ActionButton {objectName:"project-create-submit";text:"Create project & boot ships";primary:true;enabled:!!root.controller&&!root.service.busy&&!!root.connection&&Model.writable(root.connection,root.service.now)&&Model.hasFeature(root.connection,"project-member-observe-v1")&&Model.hasFeature(root.connection,"project-dependencies-v1")&&root.problem===""&&consent.checked;onClicked:{if(root.controller.start(root.connection,projectName.text.trim(),root.members,consent.checked,""))consent.checked=false}}
    Label {width:parent.width;caption:true;secondary:true;text:"Existing piers stay in their current network. Importing or adding a member to a running project will require a separate reviewed move."}
  }
  Label {width:parent.width;visible:!!root.controller&&root.controller.busy;text:root.controller?"Creating project · "+(root.controller.progress.step||"preparing")+(root.controller.progress.instance_id?" · "+root.controller.progress.instance_id:""):""}
  Label {width:parent.width;visible:!!root.controller&&root.controller.completed;text:"Project ships passed their first readiness checks. Refresh projects below to open their interfaces.";color:Design.success}
  Label {width:parent.width;visible:!!root.controller&&root.controller.error!=="";text:root.controller?root.controller.error:"";color:Design.danger}
  ActionButton {visible:!!root.controller&&!!root.controller.intent&&!root.controller.busy&&!root.controller.completed&&root.controller.error!=="";text:"Continue the same creation attempt";enabled:!!root.service&&!root.service.busy;onClicked:root.controller.resume()}
}
