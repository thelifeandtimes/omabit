import QtQuick
import QtQuick.Layouts
import Quickshell
import qs.Commons
import "Plugin" as Plugin

ShellRoot {
  PreviewService { id:service;monitorWindow:window }
  Plugin.MonitorWindow {
    id:window
    service:service
    title:"Omabit 0.4.0 · Setup Preview"
    width:1360;height:1000
    Rectangle { parent:window.contentItem;anchors.fill:parent;z:-1;color:window.color }
    page:"setup"
    visible:true
    onClosing:Qt.quit()
    onVisibleChanged:if(!visible)Qt.quit()
    Component.onCompleted:{setupPane.mode="development";if(Quickshell.env("OMABIT_PREVIEW_SMOKE")!=="1")showMaximized()}
    headerAccessory:Component {
      RowLayout {
        spacing:Style.space(8)
        Plugin.Label { text:"Remote fixture";caption:true;secondary:true }
        Plugin.SelectField {
          objectName:"preview-connection"
          implicitWidth:Style.space(170)
          model:["Online · operator","Offline","Online · observer"]
          currentIndex:["online","offline","observer"].indexOf(service.remoteMode)
          enabled:!window.setupPane.appDevelopmentPane.controller.busy
          onActivated:{service.remoteMode=["online","offline","observer"][currentIndex];Qt.callLater(function(){service.refresh()})}
        }
      }
    }
  }
  Timer {
    interval:1000;running:!!window.setupPane.appDevelopmentPane.policyView;repeat:false
    onTriggered:window.contentItem.grabToImage(function(result){
      result.saveToFile("/data/state/preview.png")
      console.log("READY: isolated policy preview")
      if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()
    })
  }
}
