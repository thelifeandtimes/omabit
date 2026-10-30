import QtQuick
import QtQuick.Window
import Quickshell
import Quickshell.Io
import "Plugin" as Plugin
ShellRoot {
 id: test
 property int commands:0
 property int refreshes:0
 function find(o,name,seen) {
  seen=seen||[];if(!o||seen.indexOf(o)>=0)return null;seen.push(o)
  if(o.objectName===name)return o
  for(var key of ["data","children","contentItem"]) {var list=o[key];if(!list)continue;if(typeof list.length!=="number")list=[list];for(var i=0;i<list.length;i++){var hit=find(list[i],name,seen);if(hit)return hit}}
  return null
 }
 Item {
  id: svc
  property bool localSetupAllowed:true
  property bool activationEnabled:true
  property bool transportStopping:false
  readonly property bool busy:controller.busy
  property bool checkingSetup:false
  property bool remoteSetupBusy:false
  function refresh(){test.refreshes++}
  function checkLocalSetup(){}
  function recordHelperExit(kind,status){}
  function themeCommand(args){test.commands++;return ["@HELPER@"].concat(args)}
 }
 Plugin.ServiceUpgrade {id:controller;service:svc}
 Window {visible:true;width:650;height:430;color:"#171a1e";Plugin.ServiceUpgradePane {id:pane;width:parent.width;updater:controller}}
 IpcHandler {
  target:"upgradeReview"
  function state():string{return JSON.stringify({busy:controller.busy,status:controller.status,error:controller.error,message:controller.message,commands:test.commands,refreshes:test.refreshes,apply_text:test.find(pane,"service-upgrade-apply").text})}
  function pressCheck():string{test.find(pane,"service-upgrade-check").clicked();return "true"}
  function pressUpdate():string{test.find(pane,"service-upgrade-apply").clicked();return "true"}
  function capture():string{pane.grabToImage(function(image){image.saveToFile("/home/review/upgrade-ui/review.png")});return "true"}
  function stop():string{Qt.callLater(Qt.quit);return "true"}
 }
}
