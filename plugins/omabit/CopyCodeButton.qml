import QtQuick
import "Model.js" as Model

ActionButton {
  id: root
  property var service
  property string runnerID: ""
  property string connectionName: ""
  property string instanceID: ""
  readonly property var connection: service ? service.connections.filter(function(c){return c.runner_id===root.runnerID && c.name===root.connectionName})[0] || null : null
  readonly property var instance: connection ? Model.instances(connection).filter(function(i){return i.id===root.instanceID})[0] || null : null
  text: compact ? "+code" : "Copy +code"
  hint: !service ? "Runner unavailable" : service.copyingCode ? "A sensitive clipboard session is active" : Model.codeUnavailable(connection,instance,service.now,service.observing) || "Copy +code to a sensitive clipboard for 30s"
  enabled: !!service && service.observing && !service.busy && !service.copyingCode && Model.canCopyCode(connection,instance,service.now)
  onClicked: service.copyCode(connection,instance)
}
