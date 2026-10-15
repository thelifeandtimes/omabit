import QtQuick
import qs.Commons

Column {
  id: root
  property var service
  spacing: Style.space(8)
  Label { text: "Is this an always-on machine?"; font.bold: true }
  Label { width: parent.width; secondary: true; text: "Choose how you use this computer. A local ship is available while its host is awake and connected." }
  Flow {
    width: parent.width; spacing: Style.space(8)
    ActionButton { objectName: "machine-always-on"; text: "Always-on machine"; primary: !!root.service && root.service.machineUse==="always-on"; onClicked: root.service.setMachineUse("always-on") }
    ActionButton { objectName: "machine-laptop"; text: "Laptop / sometimes off"; primary: !!root.service && root.service.machineUse==="laptop"; onClicked: root.service.setMachineUse("laptop") }
  }
  Label { width: parent.width; visible: !!root.service && root.service.machineUse==="laptop"; secondary: true; text: "Omabit on your laptop can manage remote Urbit runners on a VPS, workstation, or other always-on machine. You can also run a local ship here." }
}
