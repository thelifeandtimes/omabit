import QtQuick

// No application widget (or its singleton imports) exists before the matching
// generation's primary host is admitted. Multiple bar surfaces share that host.
Item {
  id: root
  required property string pluginId
  required property string artifact
  required property string role
  required property url widgetSource
  required property var collectionRegistry
  readonly property int collectionWidgetAPI: 1
  property QtObject bar: null
  property string moduleName: pluginId
  property var settings: ({})
  readonly property var host: collectionRegistry.host
  property var boundHost: null
  property bool attempted: false
  property string phase: "waiting"
  readonly property var payload: implementation.item
  readonly property bool admitted: !!boundHost && host === boundHost && host.phase === "loaded"
    && host.pluginId === pluginId && host.artifact === artifact && moduleName === pluginId
  readonly property bool opened: payload && "opened" in payload ? payload.opened : false
  readonly property bool popoutSwitchClosing: payload && "popoutSwitchClosing" in payload ? payload.popoutSwitchClosing : false
  implicitWidth: payload ? payload.implicitWidth : 0
  implicitHeight: payload ? payload.implicitHeight : 0
  enabled: admitted
  visible: admitted
  // Native Bar.qml identifies the outer slot item. The panel must still see
  // itself as its popout owner; translate only this widget's owner at the edge.
  function barValue(name, fallback) {
    return bar && bar[name] !== undefined ? bar[name] : fallback
  }
  property QtObject forwardedBar: QtObject {
    readonly property string pluginId: root.pluginId
    readonly property string moduleName: root.moduleName
    readonly property var shell: root.barValue("shell", null)
    readonly property color foreground: root.barValue("foreground", "transparent")
    readonly property color barForeground: root.barValue("barForeground", "transparent")
    readonly property color background: root.barValue("background", "transparent")
    readonly property color urgent: root.barValue("urgent", "transparent")
    readonly property string fontFamily: root.barValue("fontFamily", "")
    readonly property string position: root.barValue("position", "top")
    readonly property bool vertical: root.barValue("vertical", false)
    readonly property int barSize: root.barValue("barSize", 0)
    readonly property bool transparent: root.barValue("transparent", false)
    readonly property bool foregroundAnimationEnabled: root.barValue("foregroundAnimationEnabled", true)
    readonly property bool centerSectionRevealHeld: root.barValue("centerSectionRevealHeld", false)
    readonly property bool centerHoverRevealSuppressed: root.barValue("centerHoverRevealSuppressed", false)
    readonly property var activePopout: root.barValue("activePopout", null) === root
      ? root.payload : root.barValue("activePopout", null)
    readonly property var clickTargets: root.barValue("clickTargets", [])
    readonly property var layoutConfig: root.barValue("layoutConfig", {})
    function showTooltip(target, text) {if(root.admitted && root.bar)root.bar.showTooltip(target,text)}
    function hideTooltip(target) {if(root.bar)root.bar.hideTooltip(target)}
    function registerClickTarget(target) {if(root.admitted && root.bar)root.bar.registerClickTarget(target)}
    function unregisterClickTarget(target) {if(root.bar)root.bar.unregisterClickTarget(target)}
    function requestPopout(owner) {if(root.admitted && root.bar && owner===root.payload)root.bar.requestPopout(root)}
    function releasePopout(owner) {if(root.bar && owner===root.payload)root.bar.releasePopout(root)}
    function switchPanelFrom(owner, direction) {
      return root.admitted && root.bar && owner===root.payload
        ? root.bar.switchPanelFrom(root,direction) : false
    }
    function targetBelongsToWindow(target, window) {
      return root.bar ? root.bar.targetBelongsToWindow(target,window) : false
    }
    function moduleWidgets(id) {
      if(!root.admitted || !root.bar || id!==root.pluginId)return []
      return root.bar.moduleWidgets(id).map(function(item) {
        return item && item.collectionWidgetAPI===1 && item.pluginId===root.pluginId ? item.payload : item
      }).filter(function(item){return !!item})
    }
    function run(command) {if(root.admitted && root.bar)root.bar.run(command)}
    function setCenterHoverRevealSuppressed(value) {
      if(root.bar && (root.admitted || !value))root.bar.setCenterHoverRevealSuppressed(value)
    }
  }
  function reconcile() {
    if (attempted) {
      if (!admitted) {phase="fenced";close()}
      return
    }
    if (!host || host.phase !== "loaded") return
    attempted=true
    if (moduleName !== pluginId || host.pluginId !== pluginId || host.artifact !== artifact || !host.payload) {phase="fenced";return}
    boundHost=host
    var props={bar:bar?forwardedBar:null,moduleName:pluginId,settings:settings}
    if (role !== "dms") {
      props.service=host.payload
      props.manageIpc=Qt.binding(function(){return root.admitted})
    }
    implementation.setSource(widgetSource,props)
    implementation.active=true
  }
  function open() {if(admitted && payload && typeof payload.open === "function")payload.open()}
  function close() {if(payload && typeof payload.close === "function")payload.close()}
  function toggle() {if(admitted && payload && typeof payload.toggle === "function")payload.toggle()}
  function closeForPopoutSwitch() {if(payload && typeof payload.closeForPopoutSwitch === "function")payload.closeForPopoutSwitch()}
  // The primary's phase signal may run before this derived binding updates.
  // Close when admission itself changes so a mapped popup cannot outlive it.
  onAdmittedChanged: if(attempted && !admitted) {phase="fenced";close()}
  onHostChanged: Qt.callLater(reconcile)
  onModuleNameChanged: Qt.callLater(reconcile)
  onBarChanged: if(payload)payload.bar=bar?forwardedBar:null
  onSettingsChanged: if(admitted && payload)payload.settings=settings
  Component.onCompleted: Qt.callLater(reconcile)
  Connections {target:root.host;function onPhaseChanged(){root.reconcile()}}
  Loader {
    id:implementation
    active:false
    anchors.fill:parent
    onLoaded: {root.phase=root.admitted?"loaded":"fenced";if(!root.admitted)root.close()}
    onStatusChanged: if(status===Loader.Error)root.phase="fenced"
  }
}
