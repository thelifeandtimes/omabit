import QtQuick
import QtQuick.Window
import Quickshell
import "Plugin" as Plugin

ShellRoot {
  id: test
  property int phase: 0
  property bool captured: false
  property int caseIndex: 0
  property int ticks: 0
  property var cases: ["mismatch", "schema", "malformed", "members", "oversize", "timeout"]
  function check(value, message) { if (!value) throw new Error(message) }
  function control(object,name,seen) {
    seen=seen||[];if (!object || seen.indexOf(object)>=0) return null
    seen.push(object);if(object.objectName===name)return object
    for(var key of ["children","data","contentItem"]) {
      var children=object[key];if(!children)continue
      if(typeof children.length!=="number")children=[children]
      for(var i=0;i<children.length;i++){var found=control(children[i],name,seen);if(found)return found}
    }
    return null
  }
  Item {
    id: service
    property bool activationEnabled: true
    property bool busy:false
    property var projectCreation:null
    property var registrations:[]
    property var runnerPreferences:({})
    property bool observing: true
    property string profileName: "dev"
    property string scenario: "normal"
    property double now: Date.now()
    property var connections: [{name:"lab-a",runner_id:"runner-a",mode:"development-managed",status:"available",checked_at:new Date(now).toISOString(),snapshot:{},access_role:"observer",features:["project-reservations-v1","project-saved-status-v1"]}]
    function command(args) { return ["/usr/bin/python3","/lab/backend.py",scenario,profileName].concat(args) }
  }
  Window {
    visible: true; width: 660; height: 740; color: "#171a1e"
    Plugin.ProjectNetworksPane { id: pane; x:20; y:20; width:620; active:true; service:service }
  }
  Timer {
    interval:25; repeat:true; running:true
    onTriggered: {
      try {
        var r=pane.controller, refresh=control(pane,"projects-refresh"), history=control(pane,"projects-history")
        if(test.phase===0 && refresh.enabled) { pane.present(service.connections[0],"project-one");test.phase=1 }
        else if(test.phase===1 && !r.busy && r.projects.length===1) { test.check(history.enabled,"observer cannot read saved history");test.check(pane.selectedProject.id==="project-one","fleet navigation selected another project");test.phase=2 }
        else if(test.phase===2 && !r.busy && r.detail) {
          test.check(r.detail.operations[0].status==="interrupted","uncertainty hidden")
          test.check(control(pane,"projects-history-boundary").text.indexOf("Live ship health has not been checked")>=0,"history presented as live health")
          pane.grabToImage(function(result){test.captured=result.saveToFile("/lab/history.png")});test.phase=3
        } else if(test.phase===3 && test.captured) { service.scenario=test.cases[0];test.phase=4 }
        else if(test.phase===4 && !r.busy && refresh.enabled) { r.requestTimeoutMs=service.scenario==="timeout"?80:15000;refresh.clicked();test.phase=5 }
        else if(test.phase===5 && !r.busy && r.error!=="") {
          test.check(r.projects.length===0 && r.detail===null,"invalid response became data")
          test.caseIndex++
          if(test.caseIndex<test.cases.length) { service.scenario=test.cases[test.caseIndex];test.phase=4 }
          else { service.scenario="changed";r.requestTimeoutMs=15000;test.phase=6 }
        } else if(test.phase===6 && refresh.enabled) { refresh.clicked();test.phase=7 }
        else if(test.phase===7 && !r.busy && r.projects.length===1) { history.clicked();test.phase=8 }
        else if(test.phase===8 && !r.busy && r.error!=="") { test.check(!r.detail,"changed revision accepted");service.scenario="older";test.phase=9 }
        else if(test.phase===9 && refresh.enabled) { refresh.clicked();test.phase=10 }
        else if(test.phase===10 && !r.busy && r.projects.length===1) { history.clicked();test.phase=11 }
        else if(test.phase===11 && !r.busy && r.detail) {
          test.check(r.detail.operations.length===128 && control(pane,"projects-older").visible,"older-history truncation hidden")
          service.scenario="delay";test.phase=12
        } else if(test.phase===12 && refresh.enabled) { refresh.clicked();test.phase=13 }
        else if(test.phase===13 && !r.busy && r.projects.length===1) { history.clicked();test.ticks=0;test.phase=14 }
        else if(test.phase===14 && ++test.ticks>=6) { service.profileName="edge";test.phase=15 }
        else if(test.phase===15 && !r.busy) {
          test.check(!r.detail && !r.projects.length && !r.error,"late result crossed profile")
          service.scenario="empty";test.phase=16
        } else if(test.phase===16 && refresh.enabled) { refresh.clicked();test.phase=17 }
        else if(test.phase===17 && !r.busy && r.readAt!=="") {
          test.check(r.projects.length===0 && !history.visible,"empty project inventory misrepresented")
          var c=JSON.parse(JSON.stringify(service.connections[0]));c.features=[];service.connections=[c];test.phase=18
        } else if(test.phase===18) {
          test.check(!refresh.enabled && !r.projects.length,"unsupported runner allowed read")
          var c=JSON.parse(JSON.stringify(service.connections[0]));c.features=["project-reservations-v1"];service.connections=[c];service.scenario="normal";test.phase=19
        } else if(test.phase===19 && refresh.enabled) { refresh.clicked();test.phase=20 }
        else if(test.phase===20 && !r.busy && r.projects.length===1) {
          test.check(!history.enabled,"missing status capability ignored")
          service.observing=false;test.phase=21
        } else if(test.phase===21) {
          test.check(!refresh.enabled && !r.projects.length && !r.detail,"offline runner retained current view")
          pane.active=false;test.check(!r.canList,"inactive pane can read")
          console.log("PASS: public project UI");Qt.quit()
        }
      } catch(error) { console.error("FAIL: phase "+test.phase+" "+error.message+" "+error.stack);Qt.exit(1) }
    }
  }
  Timer { interval:30000;running:true;onTriggered:{console.error("FAIL: project UI timed out at "+test.phase);Qt.exit(1)} }
}
