import QtQuick
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  objectName: "remote-runner-setup"
  property var service
  property bool advanced: false
  property bool preparedConnection: false
  property string preparedTransport: "ssh"
  signal viewRunners()
  readonly property var setup: service ? service.remoteSetup : null
  readonly property bool blocked: !service || service.busy || service.remoteSetupBusy
  spacing: Style.space(12)
  onVisibleChanged: if (visible && service) service.checkRemoteSetup()
  Label { text: "Add a remote runner"; font.pixelSize: Style.space(22); font.bold: true }
  Label { width: parent.width; secondary: true; text: "Use the SSH alias or hostname you normally sign in to. Omabit will find an existing runner or guide you through installing one, then set up a dedicated key for this desktop." }
  Label { text: "SSH host or alias"; caption: true; secondary: true }
  SSHHostField { id: host; service: root.service; objectName: "remote-setup-host"; width: parent.width; enabled: !root.blocked; configFile:config.text; onSuggestionSelected:function(value){port.text=value ? String(value) : ""} }
  Label { width:parent.width;caption:true;secondary:true;text:"Choose from your local SSH files, or type any custom host." }
  Label { width:parent.width;caption:true;secondary:true;visible:port.text!==""&&!root.advanced;text:"SSH port "+port.text+" · change in Advanced" }
  ActionButton { objectName: "remote-setup-advanced"; text: root.advanced ? "Advanced −" : "Advanced +"; quiet: true; compact: true; onClicked: root.advanced=!root.advanced }
  Column {
    width: parent.width; spacing: Style.space(8); visible: root.advanced
    Label { text: "Connection name · optional"; caption: true; secondary: true }
    Field { id: name; objectName:"remote-setup-name"; width: parent.width; placeholderText: "Defaults to the SSH host"; enabled: !root.blocked }
    Label { text: "SSH login user and port · optional"; caption: true; secondary: true }
    RowLayout { width: parent.width
      Field { id: user; objectName:"remote-setup-user"; Layout.fillWidth:true; placeholderText:"From your SSH configuration"; enabled:!root.blocked }
      Field { id: port; objectName:"remote-setup-port"; Layout.preferredWidth:Style.space(100); placeholderText:"SSH port"; validator:IntValidator{bottom:1;top:65535} enabled:!root.blocked }
    }
    Label { text: "SSH configuration file · optional"; caption: true; secondary: true }
    Field { id: config; objectName:"remote-setup-config"; width:parent.width; placeholderText:"Uses your normal SSH configuration"; enabled:!root.blocked }
    Label { text: "Pier-owning Unix user · fresh installs only"; caption: true; secondary: true }
    Field { id: owner; objectName:"remote-setup-owner"; width:parent.width; placeholderText:"Defaults to the SSH login user; must be non-root"; enabled:!root.blocked }
    SwitchField { id: observer; objectName:"remote-setup-observer"; text:"Read-only access to an existing runner"; enabled:!root.blocked }
    ActionButton { text: root.preparedConnection ? "Hide prepared connection" : "Connect an already configured RPC endpoint…"; quiet:true;compact:true;onClicked:root.preparedConnection=!root.preparedConnection }
  }
  Label { width:parent.width;caption:true;secondary:true;text:"Continue opens a setup terminal for your SSH key/passphrase and remote administrator approval. The final connection is tested using the dedicated management key." }
  ActionButton {
    objectName:"remote-setup-continue"; text:"Continue";primary:true
    enabled:!root.blocked&&Model.validRemoteHost(host.text.trim())&&(!name.text.trim()||Model.validId(name.text.trim()))&&(!port.text.trim()||Number(port.text)>=1&&Number(port.text)<=65535)
    onClicked:root.service.startRemoteSetup({host:host.text.trim(),name:name.text.trim(),user:user.text.trim(),port:port.text.trim(),config:config.text.trim(),owner:owner.text.trim(),role:observer.checked ? "observer" : "operator"})
  }
  Card {
    width:parent.width;visible:!!root.setup&&root.setup.phase!=="idle";implicitHeight:progress.implicitHeight+Style.space(28)
    Column {
      id:progress;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(14);spacing:Style.space(8)
      CardTitle { width:parent.width;text:root.setup ? "Remote setup · "+root.setup.options.host : "" }
      Label { objectName:"remote-setup-progress";width:parent.width;text:root.setup ? root.setup.message : "";color:root.setup&&["failed","interrupted"].indexOf(root.setup.phase)>=0 ? Design.warning : Design.ink }
      Label { width:parent.width;caption:true;secondary:true;visible:!!root.setup&&root.setup.busy;text:"Follow the setup terminal to authenticate and approve the detected action. Closing that terminal preserves the attempt for Resume." }
      Flow {
        width:parent.width;spacing:Style.space(8)
        ActionButton { objectName:"remote-setup-resume";text:root.setup&&root.setup.phase==="revoked" ? "Connect again" : root.setup&&root.setup.revoking ? "Resume revocation" : "Resume setup";visible:!!root.setup&&["failed","interrupted","revoked"].indexOf(root.setup.phase)>=0;enabled:!root.blocked;onClicked:root.service.resumeRemoteSetup(root.setup.id,root.setup.phase!=="revoked"&&root.setup.revoking===true) }
        ActionButton { objectName:"remote-setup-view";text:"View runners →";visible:!!root.setup&&root.setup.phase==="connected";onClicked:root.viewRunners() }
        ActionButton { objectName:"remote-setup-revoke";text:"Revoke this desktop’s access…";quiet:true;visible:!!root.setup&&root.setup.phase==="connected";enabled:!root.blocked;onClicked:root.service.resumeRemoteSetup(root.setup.id,true) }
      }
    }
  }
  Label { objectName:"remote-setup-error";width:parent.width;color:Design.danger;visible:text!=="";text:root.service ? root.service.remoteSetupError : "" }
  Column {
    width:parent.width;spacing:Style.space(8);visible:root.advanced&&root.preparedConnection
    Label { width:parent.width;text:"For a restricted runner endpoint already provisioned with keys and host trust, including an existing port-22222 connection.";secondary:true }
    RowLayout {
      ActionButton {text:"SSH";compact:true;primary:root.preparedTransport==="ssh";onClicked:root.preparedTransport="ssh"}
      ActionButton {text:"Local socket";compact:true;primary:root.preparedTransport==="unix";onClicked:root.preparedTransport="unix"}
    }
    Field { id: preparedName;width:parent.width;placeholderText:"Connection name" }
    Field { id: preparedHost;width:parent.width;placeholderText:root.preparedTransport==="ssh" ? "Management SSH alias" : "Absolute runner socket path" }
    Field { id: preparedConfig;width:parent.width;visible:root.preparedTransport==="ssh";placeholderText:"Optional absolute SSH config path" }
    ActionButton { text:"Connect prepared endpoint";enabled:!root.blocked;onClicked:if(root.service.enroll(preparedName.text.trim(),preparedHost.text.trim(),preparedConfig.text.trim(),root.preparedTransport))root.viewRunners() }
  }
}
