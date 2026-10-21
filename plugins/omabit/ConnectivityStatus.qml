import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property var connection
  property var instance
  readonly property var compatibility: Model.liveHostCompatibility(connection,service ? service.now : Date.now())
  readonly property string requestError: service && typeof service.runtimeActionError==="function" && instance && connection ? service.runtimeActionError(connection,instance) : ""
  spacing: Style.space(8)
  Label { objectName:"connectivity-feed-error";width:parent.width;visible:text!=="";text:root.service ? root.service.feedError : "";color:Design.warning }
  Label { objectName:"runtime-status-notice";width:parent.width;text:Model.runtimeNotice(root.instance);visible:text!=="";color:root.instance && root.instance.recovery_required ? Design.danger : Design.ink }
  Label { objectName:"runtime-host-notice";width:parent.width;caption:true;color:Design.warning;visible:text!=="";text:root.compatibility && root.compatibility.restart_pending ? "Engine restart pending · see Technical details." : root.compatibility && !root.compatibility.can_mutate ? "Host checks need attention · see Technical details." : "" }
  Label { width:parent.width;caption:true;secondary:true;visible:text!=="";text:root.service ? Model.codeUnavailable(root.connection,root.instance,root.service.now,root.service.observing) : "" }
  Label { objectName:"connectivity-request-error";width:parent.width;visible:text!=="";text:root.requestError;color:Design.danger }
  ActionButton { text:"Dismiss request message";compact:true;quiet:true;visible:root.requestError!=="";onClicked:root.service.dismissRuntimeActionError(root.connection,root.instance) }
}
