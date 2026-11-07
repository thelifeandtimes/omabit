import QtQuick
import Quickshell

Item {
  id:test
  required property var window
  readonly property var service:window.service
  readonly property var agent:service.agentHelp
  property int phase:0
  property int ticks:0
  property bool captured:false
  function check(value,message){if(!value)throw new Error(message)}
  function find(o,name){if(o.objectName===name)return o;var kids=o.children||[];for(var i=0;i<kids.length;i++){var found=find(kids[i],name);if(found)return found}return null}
  Timer {
    interval:80;running:true;repeat:true
    onTriggered:{
      try {
        test.check(++test.ticks<230,"Timed out at phase "+test.phase)
        var a=test.agent,w=test.window
        if(a.busy)return
        test.check(!a.error,a.error)
        if(test.phase===0 && a.value){
          test.check(a.value.agent==="opencode" && a.target==="/data/config/opencode/skills","Default agent or installation directory incorrect")
          test.find(w.contentItem,"agent-skill-install-omabit-operator").clicked();test.phase=1
        }else if(test.phase===1 && a.value.skills[0].state==="installed"){
          test.find(w.contentItem,"agent-skill-install-omabit-plugin-developer").clicked();test.phase=2
        }else if(test.phase===2 && a.value.skills.every(function(s){return s.state==="installed"})){
          test.find(w.contentItem,"agent-help-launch").clicked();test.phase=3
        }else if(test.phase===3 && a.message.indexOf("Opened opencode")===0){
          test.service.remoteMode="offline";a.refresh();test.phase=4
        }else if(test.phase===4 && a.value.agent===""){
          test.check(a.value.problem.indexOf("Defaults")>=0,"No-default guidance missing")
          test.check(!test.find(w.contentItem,"agent-help-launch").enabled,"Launch enabled without an agent")
          w.contentItem.grabToImage(function(result){test.captured=result.saveToFile("/data/state/agent-no-default.png")})
          test.phase=5
        }else if(test.phase===5 && test.captured){
          test.service.remoteMode="online";a.refresh();test.phase=6
        }else if(test.phase===6 && a.value.agent==="opencode"){
          test.service.remoteSetup={id:"11111111111111111111111111111111",phase:"failed",message:"Fixture failure",options:{host:"fixture-only"},busy:false}
          w.setupPane.mode="remote";test.phase=7
        }else if(test.phase===7){
          var help=test.find(w.contentItem,"remote-setup-agent-help");test.check(help.visible && help.enabled,"Failed setup has no Agent Help")
          help.clicked();test.phase=8
        }else if(test.phase===8 && a.message.indexOf("Opened opencode")===0){
          w.setupPane.mode="agent";w.width=900;test.phase=9
        }else if(test.phase===9 && a.value){
          test.captured=false;w.contentItem.grabToImage(function(result){test.captured=result.saveToFile("/data/state/agent-skills.png")});test.phase=10
        }else if(test.phase===10 && test.captured){console.log("PASS: agent help uses the default launcher, preserves the profile, installs both complete skills and explains no default");Qt.quit()}
      }catch(e){console.error("FAIL: "+e.message+" "+e.stack);Qt.exit(1)}
    }
  }
}
