import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var service
  property string runnerFilter: ""
  property string shipFilter: ""
  property string kindFilter: "all"
  property string expanded: ""
  readonly property var runners: [{label:"All runners",value:""}].concat(service ? service.connections.map(function(c) { return {label:Model.runnerTitle(c,service.runnerPreferences),value:c.runner_id} }) : [])
  readonly property var ships: [{label:"All ships",value:""}].concat(service ? Model.fleet(service.connections).filter(function(s) { return !root.runnerFilter || s.connection.runner_id === root.runnerFilter }).map(function(s) { return {label:Model.instanceTitle(s.instance) + " · " + s.instance.id,value:s.key} }) : [])
  readonly property var events: Model.activity(service ? service.connections : [], runnerFilter, shipFilter, kindFilter, service ? service.runnerPreferences : {})
  spacing: Style.space(16)
  Label { text: "Activity"; font.pixelSize: Style.space(26); font.bold: true }
  Label { width: parent.width; text: "Operations and incidents across your fleet. Times are local; offline runners retain their last received history."; secondary: true }
  RowLayout {
    width: parent.width
    SelectField { objectName: "activity-runner-filter"; Layout.fillWidth: true; model: root.runners; textRole: "label"; currentIndex: Math.max(0, root.runners.findIndex(function(r) { return r.value === root.runnerFilter })); onActivated: { root.runnerFilter = root.runners[currentIndex].value; root.shipFilter = "" } }
    SelectField { objectName: "activity-ship-filter"; Layout.fillWidth: true; model: root.ships; textRole: "label"; currentIndex: Math.max(0, root.ships.findIndex(function(s) { return s.value === root.shipFilter })); onActivated: root.shipFilter = root.ships[currentIndex].value }
  }
  Flow {
    width: parent.width; spacing: Style.space(6)
    Repeater {
      model: [{key:"all",label:"Everything"},{key:"operations",label:"Operations"},{key:"incidents",label:"Incidents"},{key:"attention",label:"Needs attention"}]
      ActionButton { required property var modelData; text: modelData.label; compact: true; primary: root.kindFilter === modelData.key; onClicked: root.kindFilter = modelData.key }
    }
  }
  Label { visible: root.events.length === 0; text: "No events match these filters."; secondary: true }
  Repeater {
    model: root.events
    Card {
      id: event
      required property var modelData
      width: root.width; implicitHeight: body.implicitHeight + Style.space(20)
      Column {
        id: body
        anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; anchors.margins: Style.space(10); spacing: Style.space(6)
        RowLayout {
          width: parent.width
          ShipName { objectName: "activity-subject"; Layout.fillWidth: true; Layout.minimumWidth: 0; instance: event.modelData.subject; font.bold: true; font.pixelSize: Style.space(13) }
          Label { objectName: "activity-time"; Layout.alignment: Qt.AlignRight | Qt.AlignTop; text: Model.humanTime(event.modelData.time,root.service.now); secondary: true; caption: true }
        }
        Label { objectName: "activity-action"; width: parent.width; text: Model.operationTitle(event.modelData.kind); font.bold: true }
        RowLayout {
          width: parent.width
          ActionButton { id: details; objectName: "activity-details"; compact: true; quiet: true; text: root.expanded === event.modelData.key ? "Less −" : "Details +"; onClicked: root.expanded = root.expanded === event.modelData.key ? "" : event.modelData.key }
          Item { Layout.fillWidth: true; Layout.minimumWidth: 0 }
          ShipContext { objectName: "activity-context"; Layout.alignment: Qt.AlignRight | Qt.AlignBottom; Layout.preferredWidth: Math.min(naturalWidth,body.width-details.implicitWidth-Style.space(12)); Layout.minimumWidth: 0; host: event.modelData.runner; instanceName: event.modelData.instance || "Runner event"; status: event.modelData.status; tone: event.modelData.tone }
        }
        Label { width: parent.width; visible: text !== "" && (event.modelData.tone === "bad" || root.expanded === event.modelData.key); text: event.modelData.message; color: Design.tone(event.modelData.tone); caption: true }
        Label { width: parent.width; visible: root.expanded === event.modelData.key; text: event.modelData.detail; caption: true; secondary: true }
      }
    }
  }
  Label { text: "Showing " + root.events.length + " events · latest 300 maximum"; caption: true; secondary: true }
}
