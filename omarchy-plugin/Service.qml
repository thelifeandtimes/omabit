import QtQuick
import Quickshell
import Quickshell.Io
import "TendModel.js" as TendModel

Item {
    id: root

    property var shell: null
    property var manifest: null
    property string connectionState: "disconnected"
    property string errorMessage: ""
    property string baseUrl: ""
    property string ship: ""
    property var lists: []
    property bool mutationPending: false
    readonly property int incompleteCount: TendModel.incompleteCount(lists)
    readonly property string pluginDir: manifest && manifest.__sourceDir ? String(manifest.__sourceDir) : ""
    readonly property string bridgePath: pluginDir ? pluginDir + "/transport/eyre_client.py" : ""
    readonly property string runtimeRoot: (Quickshell.env("XDG_RUNTIME_DIR") || (Quickshell.env("HOME") + "/.cache")) + "/omabit/tend"
    readonly property string configRoot: (Quickshell.env("XDG_CONFIG_HOME") || (Quickshell.env("HOME") + "/.config")) + "/omabit/tend"
    readonly property string cookiePath: runtimeRoot + "/cookies.txt"
    readonly property string connectionPath: configRoot + "/connection.json"

    function operationId() {
        return Date.now().toString(36) + "-" + Math.floor(Math.random() * 2.14748e+09).toString(36);
    }

    function login(url, code) {
        if (!bridgePath || loginProcess.running)
            return ;

        root.errorMessage = "";
        root.connectionState = "authenticating";
        loginProcess.secret = String(code || "");
        loginProcess.command = ["python3", bridgePath, "login", "--url", String(url || ""), "--cookie", cookiePath, "--config", connectionPath];
        loginProcess.running = true;
    }

    function restore() {
        if (!bridgePath || statusProcess.running)
            return ;

        statusProcess.command = ["python3", bridgePath, "status", "--cookie", cookiePath, "--config", connectionPath];
        statusProcess.running = true;
    }

    function startStream() {
        if (!bridgePath || !ship || streamProcess.running)
            return ;

        root.connectionState = "checking";
        streamProcess.command = ["python3", bridgePath, "stream", "--cookie", cookiePath, "--config", connectionPath];
        streamProcess.running = true;
    }

    function submit(action) {
        if (connectionState !== "online" || mutationPending || pokeProcess.running)
            return false;

        root.mutationPending = true;
        root.errorMessage = "";
        pokeProcess.payload = JSON.stringify(action);
        pokeProcess.command = ["python3", bridgePath, "poke", "--cookie", cookiePath, "--config", connectionPath];
        pokeProcess.running = true;
        return true;
    }

    function createList(title) {
        return submit({
            "create-list": {
                "operation-id": operationId(),
                "title": String(title || "").trim()
            }
        });
    }

    function addReminder(listId, title, baseRevision) {
        return submit({
            "add-reminder": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "title": String(title || "").trim(),
                "base-revision": Number(baseRevision)
            }
        });
    }

    function setCompleted(listId, reminderId, completed, baseRevision) {
        return submit({
            "set-completed": {
                "operation-id": operationId(),
                "list-id": Number(listId),
                "reminder-id": Number(reminderId),
                "completed": completed === true,
                "base-revision": Number(baseRevision)
            }
        });
    }

    function handleStreamLine(line) {
        try {
            var envelope = JSON.parse(String(line || ""));
            var message = envelope.message || {
            };
            if (message.response === "subscribe" && message.ok !== undefined) {
                root.connectionState = "online";
                return ;
            }
            if (message.response !== "diff")
                return ;

            var result = TendModel.reduce(root.lists, message.json);
            root.lists = result.lists;
            if (result.error)
                root.errorMessage = result.error;

        } catch (error) {
            root.errorMessage = "Could not parse an Eyre event: " + error;
        }
    }

    function processError(raw, fallback) {
        try {
            var parsed = JSON.parse(String(raw || ""));
            return String(parsed.message || fallback);
        } catch (error) {
            return String(raw || fallback).trim();
        }
    }

    onManifestChanged: {
        if (manifest) {
            Qt.callLater(restore);
        }
    }

    Process {
        id: statusProcess

        property string output: ""
        property string errors: ""

        onExited: function(exitCode) {
            if (exitCode !== 0)
                return ;

            try {
                var status = JSON.parse(output);
                root.baseUrl = status.baseUrl || "";
                root.ship = status.ship || "";
                if (status.authenticated)
                    root.startStream();

            } catch (error) {
                root.errorMessage = "Could not read the saved Tend connection";
            }
        }

        stdout: StdioCollector {
            waitForEnd: true
            onStreamFinished: statusProcess.output = text
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: statusProcess.errors = text
        }

    }

    Process {
        id: loginProcess

        property string secret: ""
        property string output: ""
        property string errors: ""

        stdinEnabled: true
        onStarted: {
            write(secret + "\n");
            secret = "";
        }
        onExited: function(exitCode) {
            if (exitCode !== 0) {
                root.connectionState = "error";
                root.errorMessage = root.processError(errors, "Could not authenticate to Eyre");
                return ;
            }
            try {
                var result = JSON.parse(output);
                root.baseUrl = result.baseUrl;
                root.ship = result.ship;
                root.startStream();
            } catch (error) {
                root.connectionState = "error";
                root.errorMessage = "Eyre returned an invalid login response";
            }
        }

        stdout: StdioCollector {
            waitForEnd: true
            onStreamFinished: loginProcess.output = text
        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: loginProcess.errors = text
        }

    }

    Process {
        id: streamProcess

        onExited: function() {
            if (root.ship) {
                root.connectionState = "offline";
                reconnectTimer.restart();
            } else {
                root.connectionState = "disconnected";
            }
        }

        stdout: SplitParser {
            onRead: function(line) {
                root.handleStreamLine(line);
            }
        }

        stderr: SplitParser {
            onRead: function(line) {
                root.errorMessage = root.processError(line, "Eyre stream failed");
            }
        }

    }

    Process {
        id: pokeProcess

        property string payload: ""
        property string errors: ""

        stdinEnabled: true
        onStarted: {
            write(payload);
            payload = "";
        }
        onExited: function(exitCode) {
            root.mutationPending = false;
            if (exitCode !== 0)
                root.errorMessage = root.processError(errors, "Tend action failed");

        }

        stderr: StdioCollector {
            waitForEnd: true
            onStreamFinished: pokeProcess.errors = text
        }

    }

    Timer {
        id: reconnectTimer

        interval: 5000
        repeat: false
        onTriggered: root.startStream()
    }

}
