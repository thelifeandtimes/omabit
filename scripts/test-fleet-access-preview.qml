import QtQuick
import Quickshell

Item {
  id:test
  required property var window
  property int phase:0
  property int ticks:0
  property bool captured:false
  onPhaseChanged:console.log("Fleet access fixture phase "+phase)
  function check(v,message){if(!v)throw new Error(message)}
  function find(o,name){if(o.objectName===name)return o;var kids=o.children||[];for(var i=0;i<kids.length;i++){var f=find(kids[i],name);if(f)return f}return null}
  Timer {
    interval:80;running:true;repeat:true
    onTriggered:{try{
      test.check(++test.ticks<480,"Timed out at phase "+test.phase)
      var w=test.window,c=w.service.applicationConnections,p=test.find(w.contentItem,"application-connections-pane")
      if(!p || c.busy || !c.fresh)return
      var toggle=test.find(p,"apps-consent-fixture.messenger"),connect=test.find(p,"apps-connect-fixture.messenger"),select=test.find(p,"apps-ship")
      if(test.phase===0 && c.review){
        test.check(p.choices.length===3 && p.automatic,"Remote fleet ship without a local URL is missing")
        test.check(!toggle.checked && !connect.enabled,"Default permission was allowed")
        toggle.checked=true;toggle.clicked();test.phase=1
      }else if(test.phase===1 && c.review && c.grant(c.plugins[0]) && c.grant(c.plugins[0]).allowed){
        test.check(toggle.checked && connect.enabled,"Standing permission not saved")
        c.refresh();test.phase=2
      }else if(test.phase===2 && c.review){
        test.check(toggle.checked && connect.enabled,"Refresh lost standing permission")
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/fleet-permissions-wide.png")});test.phase=3
      }else if(test.phase===3 && test.captured){connect.clicked();test.phase=4
      }else if(test.phase===4 && c.review && c.attempts.length===1){
        test.check(c.attempts[0].phase==="connected" && toggle.checked,"Automatic connect failed")
        test.check(c.maintenanceCandidates().length===1,"Approved capable connection missing from maintenance")
        w.service.activationEnabled=false;test.check(!c.maintenanceTick(),"Maintenance ignored transport shutdown");w.service.activationEnabled=true
        test.check(c.maintenanceTick(),"Healthy maintenance did not start");test.phase=40
      }else if(test.phase===40 && c.review){
        test.check(c.attempts.length===1 && c.maintenanceStatus[c.attempts[0].request_id]==="healthy","Health maintenance changed connection")
        w.service.applicationScenario="switch-url";test.check(c.maintenanceTick(),"Reconnect maintenance did not start");test.phase=41
      }else if(test.phase===41 && c.review && c.attempts.length===2){
        test.check(c.attempts[0].phase==="retiring" && c.attempts[1].phase==="connected" && c.attempts[1].replacement.previous_id===c.attempts[0].request_id,"Successor lineage missing from panel")
        test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/fleet-reconnect-wide.png")});test.phase=42
      }else if(test.phase===42 && test.captured){
        w.width=760;test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/fleet-reconnect-narrow.png")});test.phase=43
      }else if(test.phase===43 && test.captured){
        w.service.applicationScenario="normal";test.check(c.maintenanceTick(),"Original cleanup maintenance did not start");test.phase=44
      }else if(test.phase===44 && c.review && c.attempts.length===1){
        test.check(c.attempts[0].origin==="https://public.preview.example" && c.attempts[0].phase==="connected","Old cleanup removed successor")
        toggle.checked=false;toggle.clicked();test.phase=5
      }else if(test.phase===5 && c.review){
        test.check(!toggle.checked && c.attempts[0].phase==="connected","Revoking permission disconnected or hid the active session")
        test.check(c.maintenanceCandidates().length===0,"Revoked connection can reconnect automatically")
        test.find(p,"apps-action-app-disconnect").clicked();test.phase=6
      }else if(test.phase===6 && c.review && c.attempts.length===0){
        test.check(!toggle.checked && !connect.enabled,"Disconnect reapproved future access")
        select.currentIndex=2;test.phase=7
      }else if(test.phase===7 && c.review && c.review.routes.record.binding.ship==="~bus"){
        test.check(!toggle.checked,"Another ship inherited permission")
        toggle.checked=true;toggle.clicked();test.phase=8
      }else if(test.phase===8 && c.review && toggle.checked){
        w.width=760;test.captured=false;w.contentItem.grabToImage(function(image){test.captured=image.saveToFile("/data/state/fleet-permissions-narrow.png")});test.phase=9
      }else if(test.phase===9 && test.captured){
        w.service.remoteMode="offline";w.service.refresh();test.phase=10
      }else if(test.phase===10){
        test.check(!c.review || c.review.routes.record.binding.ship!=="~bus","Offline ship retained review authority")
        var revoke=test.find(p,"apps-revoke-preview-remote-bus-fixture.messenger")
        test.check(revoke && revoke.enabled,"Offline permission is not revocable");revoke.clicked();test.phase=11
      }else if(test.phase===11){
        test.check(!c.permissions.entries.some(function(g){return g.allowed}),"Offline revoke failed")
        w.service.remoteMode="online";w.service.refresh();select.currentIndex=0;test.phase=12
      }else if(test.phase===12 && c.review){
        w.service.applicationScenario="conflict";c.refresh();test.phase=13
      }else if(test.phase===13 && c.review){toggle.checked=true;toggle.clicked();test.phase=14
      }else if(test.phase===14 && c.review){
        test.check(c.failed && !toggle.checked && !c.permissions.entries.some(function(g){return g.allowed}),"Conflicted allow was replayed")
        w.service.applicationScenario="normal";c.refresh();test.phase=15
      }else if(test.phase===15 && c.review){toggle.checked=true;toggle.clicked();test.phase=16
      }else if(test.phase===16 && c.review && toggle.checked){
        w.service.applicationScenario="lost-connect";c.refresh();test.phase=17
      }else if(test.phase===17 && c.review){connect.clicked();test.phase=18
      }else if(test.phase===18 && c.review && c.attempts.length===1){
        test.check(c.attempts[0].phase==="unresolved" && test.find(p,"apps-action-app-recover").enabled,"Lost reply lacks recovery")
        c.refresh();test.phase=19
      }else if(test.phase===19 && c.review){
        test.check(c.attempts.length===1 && c.attempts[0].phase==="unresolved","Refresh replayed connect")
        console.log("PASS: standing fleet permission, automatic connection, independent disconnect, offline revocation and no lost-reply replay");Qt.quit()
      }
    }catch(e){console.error("FAIL: phase "+test.phase+" "+e.message+" "+e.stack);Qt.exit(1)}}
  }
}
