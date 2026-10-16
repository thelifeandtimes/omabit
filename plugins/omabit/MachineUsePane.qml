import QtQuick
import qs.Commons

Column {
  id: root
  property var service
  spacing: Style.space(8)
  SwitchField { objectName: "machine-use-toggle"; width: parent.width; text: "Always-on machine"; checked: !!root.service && root.service.machineUse!=="laptop"; enabled: !!root.service && !root.service.settingUp; onToggled: root.service.setMachineUse(checked ? "always-on" : "laptop") }
  Label { width: parent.width; secondary: true; text: root.service && root.service.machineUse==="laptop" ? "Laptop / sometimes off. Local ships are available while this computer is awake and connected." : "This computer normally stays awake and connected." }
  Label { width: parent.width; visible: !!root.service && root.service.machineUse==="laptop"; secondary: true; text: "Omabit on your laptop can manage remote Urbit runners on a VPS, workstation, or other always-on machine. You can also run a local ship here." }
}
