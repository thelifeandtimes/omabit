import QtQuick
import QtQuick.Controls
import Quickshell
import Quickshell.Wayland
import qs.Commons

Item {
    id: root

    property var shell: null
    property var manifest: null
    property var service: null
    property bool opened: false
    property int selectedListId: 0
    readonly property var selectedList: {
        var available = service ? service.lists : [];
        for (var i = 0; i < available.length; i++) {
            if (available[i].id === selectedListId)
                return available[i];

        }
        return available.length ? available[0] : null;
    }

    function open(payloadJson) {
        root.opened = true;
        Qt.callLater(function() {
            if (service && service.ship)
                quickAdd.forceActiveFocus();
            else
                shipUrl.forceActiveFocus();
        });
    }

    function close() {
        root.opened = false;
    }

    function dismiss() {
        root.opened = false;
        if (root.shell)
            root.shell.hide("io.omabit.tend");

    }

    function toggle() {
        if (root.opened)
            root.dismiss();
        else
            root.open("{}");
    }

    onSelectedListChanged: {
        if (selectedList) {
            selectedListId = selectedList.id;
        }
    }

    PanelWindow {
        id: window

        visible: root.opened
        color: "transparent"
        WlrLayershell.namespace: "omabit-tend"
        WlrLayershell.layer: WlrLayer.Overlay
        WlrLayershell.keyboardFocus: WlrKeyboardFocus.Exclusive
        exclusionMode: ExclusionMode.Ignore

        anchors {
            top: true
            bottom: true
            left: true
            right: true
        }

        Rectangle {
            anchors.fill: parent
            color: Color.menu.scrim

            MouseArea {
                anchors.fill: parent
                onClicked: root.dismiss()
            }

            Rectangle {
                id: card

                anchors.centerIn: parent
                width: Math.min(1100, parent.width - Style.space(48))
                height: Math.min(760, parent.height - Style.space(48))
                radius: Style.cornerRadius
                color: Color.menu.background
                border.color: Color.menu.border
                border.width: Math.max(1, Style.space(1))
                Keys.onEscapePressed: root.dismiss()

                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                    }
                }

                Column {
                    anchors.fill: parent
                    anchors.margins: Style.spacing.panelPadding
                    spacing: Style.space(12)

                    Row {
                        width: parent.width
                        height: Style.space(38)

                        Text {
                            width: parent.width - closeButton.width
                            text: service && service.ship ? "Tend · ~" + service.ship : "Tend"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.heading
                            font.bold: true
                        }

                        Button {
                            id: closeButton

                            text: "Close"
                            onClicked: root.dismiss()
                        }

                    }

                    Text {
                        width: parent.width
                        visible: service && service.errorMessage !== ""
                        text: service ? service.errorMessage : ""
                        color: "#ef4444"
                        wrapMode: Text.Wrap
                        font.family: Style.font.menuFamily
                        font.pixelSize: Style.font.body
                    }

                    Column {
                        visible: !service || !service.ship
                        width: Math.min(560, parent.width)
                        anchors.horizontalCenter: parent.horizontalCenter
                        spacing: Style.space(12)

                        Text {
                            width: parent.width
                            text: "Connect your Urbit"
                            color: Color.menu.text
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.title
                            font.bold: true
                        }

                        Text {
                            width: parent.width
                            text: "Enter your ship domain and the current code printed by +code. Tend accepts planets, moons, and comets."
                            color: Color.menu.text
                            opacity: 0.72
                            wrapMode: Text.Wrap
                            font.family: Style.font.menuFamily
                            font.pixelSize: Style.font.body
                        }

                        TextField {
                            id: shipUrl

                            width: parent.width
                            placeholderText: "https://sampel-palnet.arvo.network"
                        }

                        TextField {
                            id: loginCode

                            width: parent.width
                            placeholderText: "lidlut-tabwed-pillex-ridrup"
                            echoMode: TextInput.Password
                            onAccepted: connectButton.clicked()
                        }

                        Button {
                            id: connectButton

                            text: service && service.connectionState === "authenticating" ? "Connecting…" : "Connect"
                            enabled: service && service.connectionState !== "authenticating" && shipUrl.text.trim() !== "" && loginCode.text.trim() !== ""
                            onClicked: {
                                service.login(shipUrl.text, loginCode.text);
                                loginCode.text = "";
                            }
                        }

                    }

                    Row {
                        visible: service && service.ship !== ""
                        width: parent.width
                        height: parent.height - y
                        spacing: Style.space(16)

                        Rectangle {
                            width: Style.space(230)
                            height: parent.height
                            radius: Style.cornerRadius
                            color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.05)

                            Column {
                                anchors.fill: parent
                                anchors.margins: Style.space(10)
                                spacing: Style.space(6)

                                Text {
                                    text: service ? service.connectionState.toUpperCase() : "DISCONNECTED"
                                    color: service && service.connectionState === "online" ? "#22c55e" : "#f59e0b"
                                    font.family: Style.font.menuFamily
                                    font.pixelSize: Style.font.caption
                                    font.bold: true
                                }

                                Repeater {
                                    model: service ? service.lists : []

                                    delegate: Button {
                                        required property var modelData

                                        width: parent.width
                                        text: modelData.title
                                        onClicked: root.selectedListId = modelData.id
                                    }

                                }

                                TextField {
                                    id: newList

                                    width: parent.width
                                    placeholderText: "New list"
                                    enabled: service && service.connectionState === "online" && !service.mutationPending
                                    onAccepted: {
                                        if (text.trim() && service.createList(text))
                                            text = "";

                                    }
                                }

                            }

                        }

                        Column {
                            width: parent.width - Style.space(246)
                            height: parent.height
                            spacing: Style.space(8)

                            Text {
                                text: root.selectedList ? root.selectedList.title : "Create your first list"
                                color: Color.menu.text
                                font.family: Style.font.menuFamily
                                font.pixelSize: Style.font.title
                                font.bold: true
                            }

                            TextField {
                                id: quickAdd

                                width: parent.width
                                placeholderText: root.selectedList ? "Add a reminder" : "Create a list first"
                                enabled: root.selectedList && service && service.connectionState === "online" && !service.mutationPending
                                onAccepted: {
                                    if (text.trim() && service.addReminder(root.selectedList.id, text, root.selectedList.revision))
                                        text = "";

                                }
                            }

                            ListView {
                                width: parent.width
                                height: parent.height - y
                                clip: true
                                spacing: Style.space(4)
                                model: root.selectedList ? root.selectedList.reminders : []

                                delegate: Rectangle {
                                    required property var modelData

                                    width: ListView.view.width
                                    height: Style.space(44)
                                    radius: Style.cornerRadius
                                    color: Qt.rgba(Color.menu.text.r, Color.menu.text.g, Color.menu.text.b, 0.04)

                                    Row {
                                        anchors.fill: parent
                                        anchors.margins: Style.space(8)
                                        spacing: Style.space(10)

                                        CheckBox {
                                            checked: modelData.completed
                                            enabled: service && service.connectionState === "online" && !service.mutationPending
                                            onClicked: service.setCompleted(root.selectedList.id, modelData.id, checked, root.selectedList.revision)
                                        }

                                        Text {
                                            anchors.verticalCenter: parent.verticalCenter
                                            text: modelData.title
                                            color: Color.menu.text
                                            opacity: modelData.completed ? 0.5 : 1
                                            font.family: Style.font.menuFamily
                                            font.pixelSize: Style.font.body
                                            font.strikeout: modelData.completed
                                        }

                                    }

                                }

                            }

                        }

                    }

                }

            }

        }

    }

}
