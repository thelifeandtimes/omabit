import QtQuick
import QtQuick.Layouts
import qs.Commons

Column {
  id:root
  property var service
  property bool active:false
  property bool adding:false
  property string loginId:""
  readonly property var connections:service?service.applicationConnections:null
  readonly property bool available:!!connections&&connections.fresh&&!service.busy
  readonly property var hosted:connections?connections.ships.filter(function(s){return !s.managed}):[]
  signal showApplications()
  spacing:Style.space(10)
  onActiveChanged:{code.text="";loginId="";if(active&&connections)connections.refresh()}
  Label {width:parent.width;text:"Remotely Hosted Urbits";font.pixelSize:Style.space(19);font.bold:true}
  Label {width:parent.width;secondary:true;text:"Sign in once to a ship hosted elsewhere, then approve its plugins in Application Connections."}
  Label {width:parent.width;visible:!!connections&&connections.message!=="";text:connections?connections.message:"";color:connections&&connections.failed?Design.danger:Design.ink}
  Label {width:parent.width;visible:root.hosted.length===0;secondary:true;text:"No hosted ships yet."}
  Flow {
    width:parent.width;spacing:Style.space(10)
    Repeater {
      model:root.hosted
      delegate:Card {
        id:card
        required property var modelData
        width:Math.min(root.width,Style.space(380));implicitHeight:content.implicitHeight+Style.space(24)
        Column {
          id:content;anchors.left:parent.left;anchors.right:parent.right;anchors.top:parent.top;anchors.margins:Style.space(12);spacing:Style.space(8)
          Label {width:parent.width;text:card.modelData.ship;font.bold:true}
          Label {width:parent.width;secondary:true;text:card.modelData.origin}
          Label {width:parent.width;text:card.modelData.status==="session_stored"?"Sign-in saved":"Sign-in needed"}
          ActionButton {objectName:"hosted-renew-"+card.modelData.id;text:"Sign in again";quiet:true;enabled:root.available;onClicked:root.loginId=card.modelData.id}
          RowLayout {
            width:parent.width;visible:root.loginId===card.modelData.id
            Field {id:renew;objectName:"hosted-code-"+card.modelData.id;Layout.fillWidth:true;placeholderText:"+code";echoMode:TextInput.Password;enabled:root.available;onVisibleChanged:if(!visible)text=""}
            ActionButton {objectName:"hosted-login-"+card.modelData.id;text:"Sign in";enabled:root.available&&renew.text.length>0;onClicked:if(root.connections.login(card.modelData,renew.text)){renew.text="";root.loginId=""}}
          }
        }
      }
    }
  }
  ActionButton {objectName:"hosted-add";text:root.adding?"Cancel":"Add a hosted ship";enabled:root.available;onClicked:{root.adding=!root.adding;code.text=""}}
  Column {
    width:parent.width;spacing:Style.space(10);visible:root.adding
    Field {id:identity;objectName:"hosted-identity";width:parent.width;placeholderText:"~ship-name";enabled:root.available}
    Field {id:url;objectName:"hosted-url";width:parent.width;placeholderText:"Ship URL";enabled:root.available}
    Field {id:code;objectName:"shipLoginCode";width:parent.width;placeholderText:"+code";echoMode:TextInput.Password;enabled:root.available}
    ActionButton {objectName:"hosted-enroll";text:"Add and sign in";enabled:root.available&&!!url.text.trim()&&!!identity.text.trim()&&!!code.text;onClicked:if(root.connections.enrollHosted(url.text,identity.text,code.text)){code.text="";root.adding=false}}
  }
}
