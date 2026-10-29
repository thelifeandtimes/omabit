import QtQuick

// Track the actual process exit, including readers outside Service's item tree.
// This is cooperative lifecycle accounting, not a process security boundary.
Item {
  id: root
  visible: false
  property var service: null
  property var process: null
  property string kind: "reader"
  property bool exitPending: false
  property var registeredService: null
  property bool initialized: false
  readonly property bool admitted: !service || service.activationEnabled !== false
  readonly property bool pending: exitPending || !!process && process.running

  function bindService() {
    if (registeredService === service) return
    if (registeredService && typeof registeredService.unregisterAuxiliaryReader === "function")
      registeredService.unregisterAuxiliaryReader(root)
    registeredService = service
    if (registeredService && typeof registeredService.registerAuxiliaryReader === "function")
      registeredService.registerAuxiliaryReader(root)
  }
  onServiceChanged: if (initialized) bindService()
  Component.onCompleted: { initialized = true; bindService() }
  Component.onDestruction: {
    if (registeredService && typeof registeredService.unregisterAuxiliaryReader === "function")
      registeredService.unregisterAuxiliaryReader(root)
  }
  Connections {
    target: root.process
    function onStarted() { root.exitPending = true }
    function onExited(exitCode, exitStatus) {
      if (root.registeredService && typeof root.registeredService.recordHelperExit === "function")
        root.registeredService.recordHelperExit(root.kind, exitStatus)
      root.exitPending = false
    }
  }
}
