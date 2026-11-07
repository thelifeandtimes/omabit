import QtQuick
import Quickshell
import QtTest

Item {
  id:test
  required property var window
  readonly property var pane:window.setupPane.appDevelopmentPane
  property bool reloadOnly:false
  property int phase:reloadOnly ? 100 : -5
  property real previousScroll:0
  property int ticks:0
  property bool captured:false
  property var graph:null
  TestCase { id:input;name:"PolicyTreeInput";when:false;optional:true }
  function check(v,message){if(!v)throw new Error(message)}
  function find(o,name){if(o.objectName===name)return o;var children=o.children||[];for(var i=0;i<children.length;i++){var r=find(children[i],name);if(r)return r}return null}
  function choose(name,index){var control=find(window.contentItem,name);check(!!control,name+" missing");control.currentIndex=index;control.activated(index)}
  function select(scope,ship,desk){
    var nodes=find(pane,"app-policy-tree").graph.nodes
    var node=nodes.filter(function(n){return n.scope===scope && (!ship || n.target.instance_id===ship) && (!desk || n.target.desk===desk)})[0]
    check(!!node,"tree node absent: "+scope+"/"+ship+"/"+desk)
    var button=find(pane,"policy-node-"+node.key);check(!!button,"node not clickable");button.clicked()
  }
  function capture(name){captured=false;window.contentItem.grabToImage(function(result){test.captured=result.saveToFile("/data/state/"+name+".png")})}
  function geometry(){
    ;["filter-gate","filter-value","search"].forEach(function(n){var c=find(pane,"app-policy-"+n);check(c.width>100 && (!c.contentItem || c.contentItem.width>80),"cramped control: "+n)})
  }
  Timer {
    interval:80;running:true;repeat:true
    onTriggered:{
      try {
        test.ticks++;test.check(test.ticks<250,"timed out at "+test.phase)
        var p=test.pane,c=p.controller
        if(c.busy || !p.policyView)return
        var tree=test.find(p,"app-policy-tree"),view=test.find(p,"app-policy-viewport")
        if(test.phase===100){
          test.check(p.commitValue==="allow" && p.publishValue==="allow" && p.createValue==="allow","reopened editor does not display saved global values")
          ;["commit","publish","create_desk"].forEach(function(gate){test.check(test.find(p,"app-policy-"+gate).currentIndex===1,"reopened dropdown differs from saved "+gate)})
          console.log("PASS: reopened editor displays persisted permissions");Qt.quit()
        }else if(test.phase===-5){
          test.check(view.width>0 && view.height>0,"tree viewport missing")
          tree.setZoom(0.5);test.phase=-4
        }else if(test.phase===-4){
          var leaf=tree.graph.nodes.filter(function(n){return n.target.instance_id==="preview-publisher" && n.target.desk==="base"})[0]
          var button=test.find(p,"policy-node-"+leaf.key)
          input.mouseClick(button,button.width/2,button.height/2,Qt.LeftButton,Qt.NoModifier,0)
          test.check(p.scope==="desk" && p.selectedDesk==="base","scaled desk click missed its target")
          test.select("global")
          input.mouseClick(test.find(p,"app-policy-zoom-in"))
          test.check(tree.zoom>0.5,"zoom in button does not work")
          tree.setZoom(1);view.contentX=0;view.contentY=0
          input.mouseWheel(view,80,80,0,120,Qt.NoButton,Qt.ControlModifier,0)
          test.check(Math.abs(tree.zoom-1.25)<0.001,"Ctrl+wheel did not zoom")
          test.check(Math.abs((view.contentX+80)/tree.zoom-80)<1 && Math.abs((view.contentY+80)/tree.zoom-80)<1,"pointer anchor moved during zoom")
          test.previousScroll=view.contentY
          input.mouseWheel(view,100,100,0,-120,Qt.NoButton,Qt.NoModifier,0)
          test.phase=-3
        }else if(test.phase===-3){
          test.check(view.contentY>test.previousScroll && Math.abs(tree.zoom-1.25)<0.001,"ordinary wheel must scroll without zooming")
          view.cancelFlick();tree.setZoom(2);view.contentX=0;view.contentY=0
          input.mouseDrag(view,view.width-40,view.height-40,-60,-60,Qt.LeftButton,Qt.NoModifier,0)
          test.phase=-2
        }else if(test.phase===-2){
          test.check(view.contentX>0 || view.contentY>0,"zoomed tree cannot be dragged")
          test.check(test.find(test.window.contentItem,"monitor-scroll").contentY===0,"tree drag moved the Setup page")
          view.cancelFlick();tree.forceActiveFocus();input.keyClick(Qt.Key_0,Qt.ControlModifier,0)
          test.check(tree.zoom===1,"keyboard zoom reset failed")
          input.keyClick(Qt.Key_Plus,Qt.ControlModifier,0);test.check(tree.zoom>1,"keyboard zoom in failed")
          input.keyClick(Qt.Key_Minus,Qt.ControlModifier,0);test.check(tree.zoom===1,"keyboard zoom out failed")
          tree.setZoom(99);test.check(tree.zoom===2 && !test.find(p,"app-policy-zoom-in").enabled,"maximum zoom not bounded")
          tree.setZoom(0.01);test.check(tree.zoom===0.25 && !test.find(p,"app-policy-zoom-out").enabled,"minimum zoom not bounded")
          input.mouseClick(test.find(p,"app-policy-zoom-reset"));test.check(tree.zoom===1,"zoom percentage did not reset")
          test.phase=0
        }else if(test.phase===0){
          test.check(test.window.page==="setup" && !!test.find(test.window.contentItem,"monitor-setup"),"full Setup/sidebar missing")
          test.check(p.createValue==="block" && p.commitValue==="block" && p.publishValue==="block","unsafe defaults")
          test.geometry();test.capture("global");test.phase=1
        }else if(test.phase===1 && test.captured){
          test.select("desk","preview-publisher","base")
          test.choose("app-policy-commit",2);test.choose("app-policy-publish",1)
          test.find(p,"app-policy-save").clicked();test.phase=2
        }else if(test.phase===2 && c.message){
          test.choose("app-policy-filter-value",1)
          test.graph=test.find(p,"app-policy-tree").graph
          test.check(test.graph.matches===1,"Allow filter does not isolate the desk")
          var leaf=test.graph.nodes.filter(function(n){return n.match})[0]
          test.check(leaf.target.desk==="base" && leaf.target.instance_id==="preview-publisher","wrong desk highlighted")
          var global=test.graph.nodes[0]
          test.check(global.onPath && !global.match && global.decision.value==="block","path incorrectly grants a parent")
          test.check(test.graph.nodes.filter(function(n){return n.target.instance_id==="preview-project-nec"}).every(function(n){return !n.onPath}),"sibling branch highlighted")
          test.phase=3
        }else if(test.phase===3){test.capture("allowed-base");test.phase=4
        }else if(test.phase===4 && test.captured){
          test.choose("app-policy-commit",1)
          tree.setZoom(0.75);tree.fitToView()
          test.check(p.scope==="desk" && p.selectedDesk==="base" && p.commitValue==="block" && tree.graph.matches===1,"zoom changed selection, unsaved edits or filter results")
          test.select("ship","preview-publisher")
          test.choose("app-policy-create_desk",2);test.find(p,"app-policy-save").clicked();test.phase=5
        }else if(test.phase===5 && c.message){
          test.check(p.effective.create_desk.value==="allow" && p.effective.create_desk.source==="ship","creation permission not saved")
          test.check(p.reviewDesk("brand-new"),"cannot review new desk")
          test.check(p.effective.commit.value==="block" && p.effective.publish.value==="block" && p.commitValue==="inherit" && p.publishValue==="inherit","new desk failed to inherit")
          test.check(!test.find(p,"app-policy-create_desk"),"desk can grant sibling creation")
          test.choose("app-policy-filter-gate",2);test.phase=6
        }else if(test.phase===6){
          test.graph=test.find(p,"app-policy-tree").graph
          test.check(test.graph.matches===1 && test.graph.nodes.filter(function(n){return n.match})[0].scope==="ship","creation filter should match the ship only")
          test.select("global");test.choose("preview-connection",1);test.phase=7
        }else if(test.phase===7 && !p.policyView.runners[1].status){
          test.find(p,"app-policy-save").clicked();test.phase=8
        }else if(test.phase===8 && p.policyView.global.revision===1){
          test.check(p.policyView.runners[1].rollout==="pending","offline rollout not pending")
          test.choose("preview-connection",0);test.phase=9
        }else if(test.phase===9 && p.policyView.runners[1].status){
          // An offline-at-review runner needs a new global review before apply.
          test.find(p,"app-policy-save").clicked();test.phase=10
        }else if(test.phase===10 && p.policyView.global.revision===2){
          test.check(p.policyView.runners.every(function(r){return r.rollout==="applied"}),"reviewed rollout not applied")
          test.choose("preview-connection",2);test.phase=11
        }else if(test.phase===11 && !p.policyView.runners[1].can_edit){
          test.select("ship","preview-remote-bus");test.check(!p.canSave,"observer can save")
          test.window.page="overview";test.find(test.window.contentItem,"monitor-setup").clicked();test.check(test.window.page==="setup","sidebar does not return to Setup")
          test.window.setupPane.mode="home"
          var card=test.find(test.window.contentItem,"setup-card-development")
          test.check(!!card,"Setup entry missing")
          var entry=card.children.filter(function(item){return typeof item.clicked==="function" && typeof item.text==="string"})[0]
          test.check(!!entry,"Setup entry action missing");entry.clicked()
          test.check(test.window.setupPane.mode==="development","Setup entry did not open the tree")
          test.window.width=1000;test.phase=12
        }else if(test.phase===12){
          test.geometry();input.mouseClick(test.find(p,"app-policy-zoom-fit"))
          test.check(tree.mapWidth*tree.zoom<=view.width+1 && tree.mapHeight*tree.zoom<=view.height+1,"Fit clips the fixture tree")
          test.check(view.contentX===0 && view.contentY===0,"Fit leaves an offset")
          var fit=test.find(p,"app-policy-zoom-fit"),edge=fit.mapToItem(p,fit.width,0)
          test.check(edge.x<=p.width,"zoom controls escape narrow pane")
          test.capture("narrow");test.phase=13
        }else if(test.phase===13 && test.captured){
          test.select("global");test.choose("app-policy-commit",1);test.choose("app-policy-publish",1);test.choose("app-policy-create_desk",1)
          test.find(p,"app-policy-save").clicked();test.phase=14
        }else if(test.phase===14 && p.policyView.global.revision===3){console.log("PASS: full Setup navigation, permission tree selection/highlights, create-desk inheritance, offline rollout, observer, responsive layout and real zoom/scroll/drag/keyboard input");Qt.quit()}
      }catch(e){console.error("FAIL: "+e.message+" "+e.stack);Qt.exit(1)}
    }
  }
}
