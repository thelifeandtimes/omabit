import QtQuick
import QtQuick.Window
import Quickshell
import "Plugin" as Plugin

ShellRoot {
  id:test
  property int phase:0
  property int ticks:0
  property bool captured:false
  function check(v,m){if(!v)throw new Error(m)}
  function control(o,n){if(o.objectName===n)return o;var c=o.children||[];for(var i=0;i<c.length;i++){var r=control(c[i],n);if(r)return r}return null}
  Item {
    id:service
    property bool activationEnabled:true
    property bool transportStopping:false
    property string profileName:"dev"
    property string scenario:"normal"
    function themeCommand(args){return ["/usr/bin/python3","/lab/backend.py",scenario,profileName].concat(args)}
  }
  Window {visible:true;width:540;height:1000;color:"#171a1e"
    Plugin.AppDevelopmentPane {id:pane;x:20;y:16;width:500;service:service;active:true}
  }
  Timer {interval:25;repeat:true;running:true;onTriggered:{
    try {
      var c=pane.controller,save=control(pane,"app-policy-save"),desk=control(pane,"app-policy-desk")
      if(test.phase===0 && c.value && !c.busy){
        test.check(pane.commitValue==="block" && pane.publishValue==="block","unsafe defaults")
        pane.scope="runner";pane.selectedRunner="runner-lab";pane.commitValue="allow";test.check(pane.save(),"runner save refused");test.phase=1
      }else if(test.phase===1 && !c.busy && c.value){
        test.check(pane.effective.commit.source==="runner","runner not effective")
        pane.scope="ship";pane.selectedShip="publisher";pane.commitValue="block";test.check(pane.save(),"ship save refused");test.phase=2
      }else if(test.phase===2 && !c.busy && c.value){
        pane.scope="desk";desk.text="my-app";pane.commitValue="allow";pane.publishValue="allow";test.check(pane.save(),"desk save refused");test.phase=3
      }else if(test.phase===3 && !c.busy && c.value){
        test.check(pane.effective.commit.source==="desk" && pane.effective.publish.value==="allow","desk permission lost")
        pane.scope="global";pane.commitValue="block";pane.publishValue="block";test.check(pane.save(),"global save refused");test.phase=4
      }else if(test.phase===4 && !c.busy && c.value){
        pane.scope="desk";test.check(pane.effective.commit.source==="desk","global erased override")
        pane.grabToImage(function(r){test.captured=r.saveToFile("/lab/policy.png")});test.phase=5
      }else if(test.phase===5 && test.captured){service.scenario="observer";c.refresh();test.phase=6}
      else if(test.phase===6 && !c.busy && c.value){test.check(!save.enabled,"observer can save");service.scenario="offline";c.refresh();test.phase=7}
      else if(test.phase===7 && !c.busy && c.value){test.check(!save.enabled && c.value.runners[0].rollout==="pending","offline permissions editable");service.scenario="mismatch";c.refresh();test.phase=8}
      else if(test.phase===8 && !c.busy && c.error){test.check(!c.value && !save.enabled,"wrong runner accepted");service.scenario="malformed";c.refresh();test.phase=9}
      else if(test.phase===9 && !c.busy && c.error){service.scenario="delay";c.refresh();test.ticks=0;test.phase=10}
      else if(test.phase===10 && ++test.ticks>=6){service.profileName="edge";test.phase=11}
      else if(test.phase===11 && !c.busy){test.check(!c.value && !c.error,"late reply crossed profile");pane.active=false;test.check(!c.available && !save.enabled,"inactive editor enabled");console.log("PASS: app policy editor");Qt.quit()}
    }catch(e){console.error("FAIL: phase "+test.phase+" "+e.message+" "+e.stack);Qt.exit(1)}
  }}
  Timer {interval:25000;running:true;onTriggered:{console.error("FAIL: app policy timed out at "+test.phase+" "+pane.controller.error);Qt.exit(1)}}
}
