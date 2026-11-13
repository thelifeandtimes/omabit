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
  property string policyBeforeWorkflow:""
  function check(v,m){if(!v)throw new Error(m)}
  function control(o,n){if(o.objectName===n)return o;var c=o.children||[];for(var i=0;i<c.length;i++){var r=control(c[i],n);if(r)return r}return null}
  Item {
    id:service
    property bool activationEnabled:true
    property bool transportStopping:false
    property string profileName:"dev"
    property string scenario:"normal"
    function themeCommand(args){return ["/usr/bin/python3","/lab/backend.py",(args.length ? scenario : "binding"),profileName].concat(args)}
  }
  function capture(name) {
    captured=false
    Qt.callLater(function(){preview.grabToImage(function(r){test.captured=r.saveToFile("/lab/"+name+".png")})})
  }
  function fits(o) {
    if(!o.visible)return
    if(o.objectName && (/^app-policy-(scope|runner|ship|desk|commit|publish|create_desk|filter-gate|filter-value|search|save|retry|refresh)$/.test(o.objectName) || /^app-workflow-(purpose|source|directory|publication|branch|repository|checks|instructions|save)$/.test(o.objectName))) {
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
      var w=control(pane,"app-workflow-pane"),wc=w.controller
      var sourcePane=control(pane,"app-source-pane"),sc=sourcePane.controller
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
      }else if(test.phase===5 && test.captured){
        pane.scope="ship";pane.commitValue="allow";test.check(c.discover("lab","runner-lab",pane.ship.target),"discovery refused");test.phase=50
      }else if(test.phase===50 && !c.busy){
        test.check(c.value && pane.inventory && pane.inventory.desks.length===3,"inventory absent")
        test.check(pane.commitValue==="allow","discovery erased unsaved edit")
        var graph=control(pane,"app-policy-tree").graph
        test.check(graph.nodes.some(function(n){return n.title==="%new-desk" && n.subtitle.indexOf("Installed")===0 && n.permissions.commit.value==="block"}),"discovery granted permission or missing node")
        test.capture("discovery");test.phase=51
      }else if(test.phase===51 && test.captured){
        service.scenario="discovery-mismatch";c.discover("lab","runner-lab",pane.ship.target);test.phase=52
      }else if(test.phase===52 && !c.busy){
        test.check(c.error && c.value && pane.inventory && pane.commitValue==="allow","mismatched discovery accepted or erased policy")
        service.scenario="discovery-error";c.discover("lab","runner-lab",pane.ship.target);test.phase=53
      }else if(test.phase===53 && !c.busy){
        test.check(c.error && c.value && pane.inventory && pane.commitValue==="allow","discovery failure erased edits")
        service.scenario="normal";test.policyBeforeWorkflow=JSON.stringify(c.value);test.phase=60
      }else if(test.phase===60 && wc.value && !wc.busy && !w.isDesk){
        test.check(w.draft.ship.purpose==="standard","unsafe ship purpose default")
        w.setField("purpose","distribution");w.setField("instructions","Use this fake to test releases.")
        test.check(w.save(),"ship workflow save refused");test.phase=61
      }else if(test.phase===61 && !wc.busy && wc.value){
        test.check(wc.value.ship.purpose==="distribution" && !w.dirty,"ship workflow not saved")
        test.check(pane.reviewDesk("my-app"),"workflow desk selection refused");test.phase=62
      }else if(test.phase===62 && wc.value && !wc.busy && w.isDesk){
        test.check(w.draft.desk.publication==="none" && w.draft.desk.source_name==="my-app","unsafe desk workflow default")
        w.setField("source_name","../outside");test.check(!!w.validationError && !w.save() && w.canSave,"invalid source accepted or editor disabled")
        w.setField("source_name","app-source");w.setField("publication","treaty");w.setField("branch","master")
        w.editChecks("Compile the Gall agent\n\nVerify the frontend\n");test.check(w.draft.desk.checks.length===2,"blank checks not normalized")
        w.setField("instructions","Follow the configured branch and checks before publishing.")
        test.check(w.save(),"desk workflow save refused");test.phase=63
      }else if(test.phase===63 && !wc.busy && wc.value){
        test.check(wc.value.desk.source_name==="app-source" && wc.value.desk.checks.length===2 && !w.dirty,"desk workflow not saved")
        test.check(test.policyBeforeWorkflow===JSON.stringify(c.value),"workflow changed permissions")
        test.fits(pane);test.capture("workflow");test.phase=64
      }else if(test.phase===64 && test.captured){
        w.setWorkflow("urgit");test.check(w.draft.desk.source_name==="" && !w.validationError,"Urgit retained Unix binding")
        w.setWorkflow("mounted");test.check(w.draft.desk.source_name==="my-app","Unix binding default missing")
        service.scenario="workflow-mismatch";wc.refresh();test.phase=65
      }else if(test.phase===65 && !wc.busy && wc.error){
        test.check(!wc.value && !w.canSave,"mismatched workflow accepted")
        service.scenario="normal";wc.refresh();test.phase=66
      }else if(test.phase===66 && !wc.busy && wc.value){
        test.check(wc.value.desk.source_name==="app-source","workflow refresh lost saved source")
        service.scenario="workflow-delay";wc.refresh();test.ticks=0;test.phase=67
      }else if(test.phase===67 && ++test.ticks>=6){
        service.profileName="workflow-profile";test.check(!wc.value,"profile switch kept workflow");test.phase=68
        // Policy reload supplies the selected target in the new profile.
        c.refresh()
      }else if(test.phase===68 && c.value && !wc.busy && wc.value){
        test.check(wc.value.desk.instructions==="workflow-profile","late workflow crossed profile")
        service.scenario="normal";service.profileName="dev";c.refresh();test.phase=69
      }else if(test.phase===69 && c.value && !c.busy && wc.value && !wc.busy){
        if(!sc.value || sc.busy)return
        test.check(sc.value.state==="synchronized","source default fixture not read")
        service.scenario="source-changes";sc.refresh();test.phase=70
      }else if(test.phase===70 && !sc.busy && sc.value){
        test.check(sc.value.state==="unix-changes" && sc.value.difference_count===2,"uncommitted source alert missing")
        sourcePane.expanded=true;test.capture("source-changes");test.phase=71
      }else if(test.phase===71 && test.captured){
        service.scenario="source-error";sc.refresh();test.phase=72
      }else if(test.phase===72 && !sc.busy && sc.error){
        test.check(sc.value && sc.value.difference_count===2 && control(pane,"app-source-summary").text==="Source status unavailable","failed comparison cleared pending work or appears clean")
        service.scenario="source-mismatch";sc.refresh();test.phase=73
      }else if(test.phase===73 && !sc.busy && sc.error){
        test.check(sc.value && sc.value.request.selection.target.instance_id==="publisher","source target mismatch accepted")
        service.scenario="source-delay";sc.refresh();test.ticks=0;test.phase=74
      }else if(test.phase===74 && ++test.ticks>=6){
        pane.active=false;test.check(!sc.value && !sc.available,"inactive pane retained live status");test.phase=75
      }else if(test.phase===75 && !sc.busy){
        test.check(!sc.value && !sc.error,"late source reply accepted after deactivation")
        service.scenario="normal";pane.active=true;test.phase=76
      }else if(test.phase===76 && c.value && !c.busy && wc.value && !wc.busy && sc.value && !sc.busy){
        service.scenario="observer";c.refresh();test.check(c.inventories.length===0,"refresh retained stale discovery");test.phase=6
      }
      else if(test.phase===6 && !c.busy && c.value){test.check(!save.enabled && !w.canSave,"observer can save");service.scenario="offline";c.refresh();test.phase=7}
      else if(test.phase===7 && !c.busy && c.value){test.check(!save.enabled && c.value.runners[0].rollout==="pending","offline permissions editable");service.scenario="mismatch";c.refresh();test.phase=8}
      else if(test.phase===8 && !c.busy && c.error){test.check(!c.value && !save.enabled,"wrong runner accepted");service.scenario="malformed";c.refresh();test.phase=9}
      else if(test.phase===9 && !c.busy && c.error){service.scenario="delay";c.refresh();test.ticks=0;test.phase=10}
      else if(test.phase===10 && ++test.ticks>=6){service.profileName="edge";test.phase=11}
      else if(test.phase===11 && !c.busy){test.check(!c.value && !c.error,"late reply crossed profile");pane.active=false;test.check(!c.available && !save.enabled,"inactive editor enabled");console.log("PASS: app policy editor");Qt.quit()}
    }catch(e){console.error("FAIL: phase "+test.phase+" "+e.message+" "+e.stack);Qt.exit(1)}
  }}
  Timer {interval:25000;running:true;onTriggered:{console.error("FAIL: app policy timed out at "+test.phase+" "+pane.controller.error);Qt.exit(1)}}
}
