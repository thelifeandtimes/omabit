import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import Quickshell
import qs.Commons
import "Plugin" as Plugin

ShellRoot {
  Item {
    id:service
    property bool activationEnabled:true
    property bool transportStopping:false
    property string remoteMode:"online"
    function themeCommand(args) {
      return ["/usr/bin/python3", "-B", "/preview/backend.py", remoteMode].concat(args)
    }
  }
  Window {
    id:window
    visible:true
    title:"Omabit 0.4.0 · Setup Preview"
    width:740;height:840;minimumWidth:360;minimumHeight:480
    color:Plugin.Design.background
    onClosing:Qt.quit()
    Rectangle {
      id:frame
      anchors.fill:parent;color:Plugin.Design.background
      ColumnLayout {
        anchors.fill:parent;spacing:0
        Column {
          Layout.fillWidth:true;Layout.margins:Style.space(20)
          spacing:Style.space(10)
          RowLayout {
            width:parent.width
            Plugin.CardTitle { Layout.fillWidth:true;Layout.minimumWidth:0;text:"Omabit · Setup" }
            Plugin.StatusBadge { text:"0.4.0 Preview";tone:"accent" }
          }
          Plugin.Label { width:parent.width;secondary:true;caption:true;text:"Simulated runners only. Changes are saved in this preview." }
          RowLayout {
            width:parent.width;spacing:Style.space(12)
            Column {
              Layout.fillWidth:true;Layout.minimumWidth:0;spacing:Style.space(6)
              Plugin.Label { width:parent.width;text:"preview-remote connection";secondary:true;caption:true }
              Plugin.SelectField {
                objectName:"preview-connection"
                width:parent.width;model:["Online · operator","Offline","Online · observer"]
                currentIndex:["online","offline","observer"].indexOf(service.remoteMode)
                enabled:!pane.controller.busy
                onActivated:{
                  service.remoteMode=["online","offline","observer"][currentIndex]
                  Qt.callLater(function(){pane.controller.refresh()})
                }
              }
            }
            Column {
              Layout.preferredWidth:Style.space(100);spacing:Style.space(6)
              Plugin.Label { width:parent.width;text:"Palette";secondary:true;caption:true }
              Plugin.SelectField { width:parent.width;model:["Dark","Light"];currentIndex:Color.light ? 1 : 0;onActivated:Color.light=currentIndex===1 }
            }
          }
        }
        Rectangle { Layout.fillWidth:true;height:1;color:Plugin.Design.line }
        Controls.ScrollView {
          id:scroll
          Layout.fillWidth:true;Layout.fillHeight:true;clip:true
          contentWidth:availableWidth
          contentHeight:pane.implicitHeight+Style.space(40)
          Controls.ScrollBar.horizontal.policy:Controls.ScrollBar.AlwaysOff
          Plugin.AppDevelopmentPane {
            id:pane
            x:Style.space(20);y:Style.space(20)
            width:scroll.availableWidth-Style.space(40)
            service:service;active:true
          }
        }
      }
    }
    // Compact visual evidence for this exact wrapper. This captures only our
    // own window, never the desktop, and does not mutate the fixture policies.
    Timer {
      interval:750;running:!!pane.policyView;repeat:false
      onTriggered:frame.grabToImage(function(result) {
        result.saveToFile("/data/state/preview.png")
        console.log("READY: isolated policy preview")
        if(Quickshell.env("OMABIT_PREVIEW_SMOKE")==="1")Qt.quit()
      })
    }
  }
}
