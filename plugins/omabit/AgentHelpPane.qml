import QtQuick
import QtQuick.Layouts
import qs.Commons

Column {
  id:root
  property var service
  readonly property var controller:service && service.agentHelp ? service.agentHelp : null
  readonly property var status:controller ? controller.value : null
  spacing:Style.space(12)
  onVisibleChanged:if(visible && controller)controller.refresh()
  Label {text:"Agent Help and skills";font.pixelSize:Style.space(19);font.bold:true}
  Label {width:parent.width;text:"Use your default Omarchy agent for help with Omabit. The help button includes the instructions matching this desktop; installing skills also makes them discoverable in other agent sessions.";secondary:true}
  Card {
    width:parent.width;implicitHeight:agentBody.implicitHeight+Style.space(32)
    Column {
      id:agentBody;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(10)
      CardTitle {text:"Default agent"}
      Label {width:parent.width;text:root.status ? root.status.agent || "No agent selected" : "Refresh to check Omarchy's default agent."}
      Label {width:parent.width;visible:!!root.status && !!root.status.problem;text:root.status ? root.status.problem : "";secondary:true}
      Flow {
        width:parent.width;spacing:Style.space(8)
        ActionButton {objectName:"agent-help-launch";text:"Open Agent Help";primary:true;enabled:!!root.status && !!root.status.agent && !root.controller.busy;onClicked:root.service.requestAgentHelp("general")}
        ActionButton {objectName:"agent-help-refresh";text:"Refresh";enabled:!!root.controller && !root.controller.busy;onClicked:root.controller.refresh()}
      }
    }
  }
  Card {
    width:parent.width;implicitHeight:skillsBody.implicitHeight+Style.space(32)
    Column {
      id:skillsBody;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(16);spacing:Style.space(10)
      CardTitle {text:"Install bundled skills"}
      Label {width:parent.width;visible:!!root.status;text:root.status ? "From Omabit "+root.status.bundle.version+(root.status.bundle.development ? " · "+root.status.bundle.development.label : "") : "";secondary:true;caption:true}
      Label {width:parent.width;text:"Agent skill directory";caption:true;secondary:true}
      Field {id:target;objectName:"agent-skills-target";width:parent.width;text:root.controller ? root.controller.target : "";placeholderText:"Absolute path to your agent's skills directory";enabled:!!root.controller && !root.controller.busy}
      Label {width:parent.width;text:"OpenCode's directory is detected automatically. For another agent, choose its documented skill directory. Existing custom or edited skills are preserved.";secondary:true;caption:true}
      ActionButton {objectName:"agent-skills-review";text:"Review destination";enabled:!!root.controller && !root.controller.busy && target.text.trim().startsWith("/");onClicked:{root.controller.target=target.text.trim();root.controller.refresh()}}
      Repeater {
        model:root.status && target.text===root.status.target ? root.status.skills : []
        Column {
          required property var modelData
          width:skillsBody.width;spacing:Style.space(8)
          Rectangle {width:parent.width;height:1;color:Design.line}
          Label {width:parent.width;font.bold:true;text:modelData.name==="omabit-operator" ? "Omabit operator" : "Omabit plugin developer"}
          Label {width:parent.width;caption:true;secondary:true;text:modelData.name==="omabit-operator" ? "Manage ships, diagnose setup, and follow the fleet's operational policies." : "Develop Urbit applications and Omarchy interfaces using the supported workflows."}
          Label {width:parent.width;caption:true;secondary:true;text:"Destination: "+modelData.target}
          Label {width:parent.width;caption:true;visible:!!modelData.message;text:modelData.message || "";color:Design.warning}
          ActionButton {objectName:"agent-skill-install-"+parent.modelData.name;text:parent.modelData.action==="update" ? "Update skill" : parent.modelData.state==="installed" ? "Installed" : parent.modelData.state==="preserved" ? "Existing skill preserved" : "Install skill";enabled:!!root.controller && !root.controller.busy && parent.modelData.action!=="none";onClicked:root.controller.install(parent.modelData)}
        }
      }
    }
  }
  Label {width:parent.width;visible:!!root.controller && !!root.controller.error;text:root.controller ? root.controller.error : "";color:Design.danger;caption:true}
  Label {width:parent.width;visible:!!root.controller && !!root.controller.message;text:root.controller ? root.controller.message : "";color:Design.success;caption:true}
}
