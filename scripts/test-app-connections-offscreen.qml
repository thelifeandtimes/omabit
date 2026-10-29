import QtQuick
import QtQuick.Controls
import Quickshell
import "Plugin" as Plugin

ShellRoot {
  id:test
  property int phase:0
  property bool captured:false
  function check(value,message){if(!value)throw new Error(message)}
  function control(object,name,seen){
    seen=seen||[];if(!object||seen.indexOf(object)>=0)return null
    seen.push(object);if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]){var children=object[key];if(!children)continue;if(typeof children.length!=="number")children=[children];for(var i=0;i<children.length;i++){var found=control(children[i],name,seen);if(found)return found}}
    return null
  }
  Item {
    id:service
    property bool activationEnabled:true
    readonly property bool busy:connections.busy
    readonly property alias applicationConnections:connections
    function command(args){return ["/usr/bin/python3",@@HELPER@@].concat(args)}
    Plugin.ApplicationConnections{id:connections;service:service}
  }
  ApplicationWindow {
    id:window
    visible:true;width:440;height:780;color:"#171a1e"
    ScrollView {
      anchors.fill:parent;anchors.margins:20;contentWidth:availableWidth
      Column {
        width:parent.width
        Plugin.ApplicationsPane{id:pane;width:parent.width;service:service;active:false;visible:active;onShowHosted:{active=false;hosted.active=true}}
        Plugin.HostedUrbitsPane{id:hosted;width:parent.width;service:service;active:false;visible:active;onShowApplications:{active=false;pane.active=true}}
      }
    }
  }
  Timer {
    interval:60;running:true;repeat:true
    onTriggered:{
      try {
        var consent=control(pane,"apps-consent-fixture.theme.auth-lab")
        var connect=control(pane,"apps-connect-fixture.theme.auth-lab")
        if(test.phase===0){
          test.check(!connections.busy&&!connections.fresh,"inactive panes started work")
          pane.active=true;test.phase=1
        }else if(test.phase===1&&connections.fresh&&!service.busy){
          test.check(connections.plugins.length===2&&connections.ships.length===1,"discovery incomplete")
          test.check(!connect.enabled&&!consent.enabled,"unauthenticated approval enabled")
          test.check(control(pane,"shipLoginCode")===null,"credentials remain in Application Connections")
          test.check(!connections.connect(connections.plugins[0],connections.ships[0],false),"missing consent accepted")
          pane.showHosted();test.phase=10
        }else if(test.phase===10&&connections.fresh&&!service.busy){
          control(hosted,"shipLoginCode").text="fixture-only-secret"
          control(hosted,"hosted-login").clicked()
          test.check(control(hosted,"shipLoginCode").text==="","code retained in field")
          test.phase=11
        }else if(test.phase===11&&connections.fresh&&!service.busy){hosted.showApplications();test.phase=2
        }else if(test.phase===2&&connections.fresh&&!service.busy){
          test.check(consent.enabled&&!connect.enabled,"consent not required after login")
          consent.checked=true;consent.clicked();test.check(connect.enabled,"explicit approval not enabled")
          pane.active=false;pane.active=true;test.phase=19
        }else if(test.phase===19&&connections.fresh&&!service.busy){
          test.check(!consent.checked&&!connect.enabled,"approval survived leaving the pane")
          pane.requestedPlugin="fixture.theme.auth-lab"
          pane.grabToImage(function(result){test.captured=result.saveToFile(@@SNAPSHOT@@)})
          test.phase=20
        }else if(test.phase===20&&test.captured){
          consent.checked=true;consent.clicked();test.check(connect.enabled,"valid approval disabled");connect.clicked()
          test.check(!connections.fresh,"approval review reused");test.phase=3
        }else if(test.phase===3&&connections.fresh&&!service.busy){
          test.check(connections.attempts.length===1&&connections.attempts[0].phase==="connected","connection missing")
          var disconnect=control(pane,"apps-action-app-disconnect")
          test.check(disconnect&&disconnect.enabled,"disconnect missing");disconnect.clicked();test.phase=4
        }else if(test.phase===4&&connections.fresh&&!service.busy){
          test.check(connections.failed&&connections.attempts[0].phase==="revoking","uncertain disconnect hidden")
          var recover=control(pane,"apps-action-app-recover")
          test.check(recover&&recover.enabled,"manual recovery missing");recover.clicked();test.phase=5
        }else if(test.phase===5&&connections.fresh&&!service.busy){
          test.check(connections.attempts[0].phase==="closed"&&connections.attempts[0].receiver_reconciliation,"receiver cleanup omitted")
          var reconcile=control(pane,"apps-action-app-reconcile")
          test.check(reconcile&&reconcile.enabled,"reconcile missing");reconcile.clicked();test.phase=6
        }else if(test.phase===6&&connections.fresh&&!service.busy){
          test.check(connections.attempts.length===0,"completed disconnect was not archived")
          test.check(connections.ships[0].status==="session_stored","parent sign-in lost")
          test.check(!connect.enabled,"disconnect approved reconnect automatically")
          console.log("PASS: actual application connection panel");Qt.quit()
        }
      }catch(error){console.error("FAIL: phase "+test.phase+" "+error.message+" "+error.stack);Qt.exit(1)}
    }
  }
  Timer{interval:30000;running:true;onTriggered:{console.error("FAIL: application panel timed out at "+test.phase);Qt.exit(1)}}
}
