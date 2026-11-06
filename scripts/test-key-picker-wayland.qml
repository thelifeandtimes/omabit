import QtQuick
import Quickshell
import Quickshell.Io
import qs.Ui as Ui
import "Plugin" as Plugin

ShellRoot {
  id: test
  property var pointerEvents: []
  Connections {
    id: browseEvents
    target: null
    function onPressed() { test.pointerEvents.push("pressed") }
    function onReleased() { test.pointerEvents.push("released") }
    function onClicked() { test.pointerEvents.push("clicked") }
    function onCanceled() { test.pointerEvents.push("canceled") }
  }
  Item { id: anchor }
  Window {
    id: background
    title: "Picker fixture background"; visible: true; width: 900; height: 700
    TextInput { id: backgroundInput; anchors.fill: parent; focus: true }
  }
  QtObject {
    id: service
    property bool busy: false
    property bool observing: true
    property double now: Date.now()
    property var registrationOutcome: null
    property var connections: []
    property string recommendedPillUrl: "https://example.invalid/pill"
    property int submissions: 0
    function createFromKey() { submissions++; throw new Error("Browsing submitted a key") }
  }
  Ui.KeyboardPanel {
    id: menu
    anchorItem: anchor; bar: null; screen: Quickshell.screens[0]
    contentWidth: 590; contentHeight: 690; focusTarget: scope
    FocusScope {
      id: scope
      anchors.fill: parent
      Keys.onEscapePressed: menu.close()
      Flickable {
        id: scroll
        anchors.fill: parent; clip: true; contentHeight: form.height
        Plugin.KeyBootPane { id: form; width: parent.width; service: service; connection: null; keyFile: "/previous/key.key" }
      }
    }
  }
  function find(object, name, seen) {
    seen=seen || []
    if(!object || seen.indexOf(object)>=0)return null
    seen.push(object)
    if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue
      if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var match=find(children[i],name,seen);if(match)return match}
    }
    return null
  }
  IpcHandler {
    target: "picker-fixture"
    function openMenu(): string {
      browseEvents.target=find(form,"key-file-browse")
      find(form,"key-file-picker").currentFolder=Qt.resolvedUrl("files/")
      menu.open=true
      return String(menu.open)
    }
    function hide(): void { menu.open=false }
    function revealBrowse(): string {
      var button=find(form,"key-file-browse")
      scroll.contentY=Math.max(0,Math.min(scroll.contentHeight-scroll.height,button.mapToItem(form,0,0).y-scroll.height/2))
      return String(scroll.contentY)
    }
    function status(): string {
      var picker=find(form,"key-file-picker")
      return JSON.stringify({open:menu.open,visible:picker.visible,path:form.keyFile,
        browseEnabled:find(form,"key-file-browse").enabled,submissions:service.submissions,
        backgroundText:backgroundInput.text,backing:menu.backingWindowVisible,
        menuVisible:menu.visible,screen:menu.screen ? menu.screen.name : "",hasWindow:!!form.Window.window,
        pointerEvents:test.pointerEvents,browseHovered:find(form,"key-file-browse").hovered})
    }
    function rectangle(name: string): string {
      var item=find(form.Window.window ? form.Window.window.contentItem : menu,name)
      if(!item || !item.visible)return "null"
      var p=item.mapToGlobal(item.width/2,item.height/2)
      return JSON.stringify({x:p.x,y:p.y,width:item.width,height:item.height,fileName:item.fileName || ""})
    }
    function quit(): void { Qt.quit() }
  }
}
