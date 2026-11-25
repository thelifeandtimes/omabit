import QtQuick
import qs.Commons
import "Model.js" as Model

Column {
  id: root
  property var connection
  property string selectedProfile: ""
  readonly property var choices: Model.runtimeChoices(connection,selectedProfile)
  signal edited()
  spacing:Style.space(6)
  Label { text:"Vere runtime";font.bold:true }
  SelectField {
    id:choice;objectName:"runtime-profile-choice";width:parent.width
    model:root.choices;textRole:"label"
    currentIndex:Math.max(0,root.choices.findIndex(function(p){return p.id===root.selectedProfile}))
    enabled:Model.hasFeature(root.connection,"runtime-profiles-v1")
    onActivated:function(index){root.selectedProfile=root.choices[index].id;root.edited()}
  }
  Label { width:parent.width;caption:true;secondary:true;text:"The selected version stays pinned until you explicitly change it. Channel builds appear after installation on this runner." }
}
