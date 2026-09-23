import QtQuick
import qs.Commons
import qs.Ui

Item {
  id: root

  property var model: []
  property string label: ""
  property bool showLabel: label !== ""
  property string textRole: ""
  property string valueRole: ""
  property int currentIndex: model && model.length ? 0 : -1
  property string displayText: ""
  readonly property int count: model && model.length ? model.length : 0
  readonly property string currentText: currentIndex >= 0 && currentIndex < count ? labelFor(model[currentIndex]) : ""
  readonly property var currentValue: currentIndex >= 0 && currentIndex < count ? valueFor(model[currentIndex]) : null
  signal activated(int index)

  function labelFor(entry) {
    if (textRole && entry && typeof entry === "object" && entry[textRole] !== undefined)
      return String(entry[textRole])
    return String(entry === undefined || entry === null ? "" : entry)
  }

  function valueFor(entry) {
    if (valueRole && entry && typeof entry === "object" && entry[valueRole] !== undefined)
      return entry[valueRole]
    return entry
  }

  function optionsForModel() {
    var result = []
    for (var i = 0; i < count; i++)
      result.push({ value: String(i), label: labelFor(model[i]) })
    return result
  }

  function chooseIndex(index) {
    if (index < 0 || index >= count)
      return
    currentIndex = index
    activated(index)
  }

  implicitWidth: Style.spacing.dropdownWidth
  implicitHeight: control.implicitHeight

  Dropdown {
    id: control
    anchors.fill: parent
    label: root.label
    showLabel: root.showLabel
    options: root.optionsForModel()
    value: root.currentIndex >= 0 ? String(root.currentIndex) : ""
    onChanged: function(value) { root.chooseIndex(Number(value)) }
  }
}
