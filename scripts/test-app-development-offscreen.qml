import QtQuick
import QtQuick.Window
import Quickshell
import qs.Commons
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
  function capture(name) {
    captured=false
    Qt.callLater(function(){preview.grabToImage(function(r){test.captured=r.saveToFile("/lab/"+name+".png")})})
  }
  function fits(o) {
    if(!o.visible)return
    if(o.objectName && /^app-policy-(scope|runner|ship|desk|commit|publish|create_desk|filter-gate|filter-value|search|save|retry|refresh)$/.test(o.objectName)) {
      var pos=o.mapToItem(pane,0,0)
      check(o.width>0 && pos.x>=-1 && pos.x+o.width<=pane.width+1,"control overflows: "+o.objectName)
    }
    var children=o.children||[];for(var i=0;i<children.length;i++)fits(children[i])
  }
  Window {visible:true;width:preview.width;height:preview.height;color:Plugin.Design.background
    Rectangle {
      id:preview
      width:Number(Quickshell.env("OMABIT_POLICY_WIDTH"));height:pane.implicitHeight+Style.space(48);color:Plugin.Design.background
      Plugin.AppDevelopmentPane {id:pane;x:Style.space(24);y:Style.space(24);width:parent.width-Style.space(48);service:service;active:true}
    }
  }
  Timer {interval:25;repeat:true;running:true;onTriggered:{
    try {
      var c=pane.controller,save=control(pane,"app-policy-save"),desk=control(pane,"app-policy-desk")
      if(test.phase===0 && c.value && !c.busy){
        test.check(pane.commitValue==="block" && pane.publishValue==="block" && pane.createValue==="block","unsafe defaults")
        test.fits(pane);test.capture("global");test.phase=100
      }else if(test.phase===100 && test.captured){
        pane.scope="runner";pane.selectedRunner="runner-lab";pane.commitValue="allow";test.check(pane.save(),"runner save refused");test.phase=1
      }else if(test.phase===1 && !c.busy && c.value){
        test.check(pane.effective.commit.source==="runner","runner not effective");test.check(c.message==="Permissions saved.","save confirmation missing")
        pane.scope="ship";pane.selectedShip="publisher";pane.commitValue="block";test.check(pane.save(),"ship save refused");test.phase=2
      }else if(test.phase===2 && !c.busy && c.value){
        test.check(pane.reviewDesk("my-app"),"desk review refused");pane.commitValue="allow";pane.publishValue="allow";test.check(pane.save(),"desk save refused");test.phase=3
      }else if(test.phase===3 && !c.busy && c.value){
        test.check(pane.effective.commit.source==="desk" && pane.effective.publish.value==="allow","desk permission lost")
        pane.scope="global";pane.commitValue="block";pane.publishValue="block";test.check(pane.save(),"global save refused");test.phase=4
      }else if(test.phase===4 && !c.busy && c.value){
        pane.scope="desk";test.check(pane.effective.commit.source==="desk","global erased override")
        test.ticks=0;test.phase=40
      }else if(test.phase===40 && ++test.ticks>=3){test.fits(pane);test.capture("policy");test.phase=5
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
