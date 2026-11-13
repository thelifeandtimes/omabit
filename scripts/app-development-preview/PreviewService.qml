import QtQuick

// Full-panel presentation adapter. No runner, collector, SSH or host operations.
Item {
  id:root
  property bool activationEnabled:true
  property bool transportStopping:false
  property bool busy:false
  property bool observing:true
  property bool firstRun:false
  property bool localSetupAllowed:false
  property bool checkingSetup:false
  property bool checkingPort:false
  property bool settingUp:false
  property bool remoteSetupBusy:false
  property bool copyingCode:false
  property bool canConfirmPrune:false
  property bool actionProgressVisible:false
  property string applicationVersion:"0.4.0 preview"
  property string releaseLabel:"SIMULATED RUNNERS"
  property string distributionChannel:"develop"
  property string profileName:"preview"
  property string machineUse:"always-on"
  property string remoteMode:"online"
  property string currentAction:""
  property string lastError:""
  property string lastResult:""
  property string feedError:""
  property string portError:""
  property string localSetupError:""
  property string localSetupMessage:""
  property string localSetupStep:""
  property string remoteSetupError:""
  property string requestIdentity:""
  property string recommendedPillUrl:""
  property string toastText:""
  property string toastTone:"muted"
  property string collector:"/preview/no-collector"
  property double now:Date.now()
  property var settings:({})
  property var bootstrap:null
  property var runnerPreferences:({})
  property var memoryHistory:({})
  property var cpuHistory:({})
  property var importOutcomes:({})
  property var inspections:({})
  property var bootTarget:null
  property var portSuggestion:null
  property var portTarget:null
  property var pendingFirstBoot:null
  property var epochPreview:null
  property var externalImportPreview:null
  property var importCatalogue:null
  property var importRecoveryReview:null
  property var hostMoveOutcome:null
  property var registrationOutcome:null
  property var storageOutcome:null
  property var routineOutcome:null
  property var remoteSetup:null
  property var monitorWindow:null
  property var localSetupStatus:({ready:true,host_present:false,connection:{name:"preview-local"}})
  property var connections:inventory()
  property var registrations:connections.map(function(r){return {name:r.name,runner_id:r.runner_id,transport:"unix",socket:"/preview/fixture-only",revision:1}})
  function inventory() {
    function ship(id,name){return {id:id,ship:name,label:id,config_revision:1,applied_revision:1,mode:"fake",config:{http_port:0,ames_port:0,loom:31},observed_state:"stopped",desired_state:"stopped",observed_at:new Date().toISOString()}}
    return [{name:"preview-local",runner_id:"preview-local-040",mode:"simulation",status:"available",access_role:"operator",checked_at:new Date().toISOString(),features:[],snapshot:{instances:[ship("preview-publisher","~zod"),ship("preview-project-nec","~nec")],operations:[],incidents:[]}},
            {name:"preview-remote",runner_id:"preview-remote-040",mode:"simulation",status:remoteMode==="offline"?"unavailable":"available",access_role:remoteMode==="observer"?"observer":"operator",checked_at:new Date().toISOString(),features:[],snapshot:{instances:[ship("preview-remote-bus","~bus")],operations:[],incidents:[]}}]
  }
  function themeCommand(args){return ["/usr/bin/python3","-B","/preview/backend.py",remoteMode].concat(args)}
  function command(args){return ["/usr/bin/python3","-B","/preview/backend.py",remoteMode,"preview-unavailable"]}
  function refresh(){now=Date.now();connections=inventory();if(monitorWindow && monitorWindow.setupPane.appDevelopmentPane.active)monitorWindow.setupPane.appDevelopmentPane.controller.refresh()}
  function unavailable(){lastError="This isolated preview supports application-policy editing only.";toastText=lastError;return false}
  function saveDesktopView(){}
  function recordHelperExit(){}
  function canConfigure(){return false}
  function canImport(){return false}
  function canReviewTarget(){return false}
  function canSaveRoutine(){return false}
  function runtimeActionError(){return ""}
  function dismissRuntimeActionError(){}
  function clearEpochPreview(){}
  function openMonitor(){return unavailable()}
  function openShip(){return unavailable()}
  function setMachineUse(value){machineUse=value}
  function checkLocalSetup(){return unavailable()}
  function checkRemoteSetup(){return unavailable()}
  function operate(){return unavailable()}
  function maintain(){return unavailable()}
  function configure(){return unavailable()}
  function copyCode(){return unavailable()}
  function submit(){return unavailable()}
  function enroll(){return unavailable()}
  function editRegistration(){return unavailable()}
  function saveRunnerPresentation(){return unavailable()}
  function updateRunner(){return unavailable()}
  function acceptDevelopmentMode(){return unavailable()}
  function createFromKey(){return unavailable()}
  function deletePier(){return unavailable()}
  function moveHost(){return unavailable()}
  function movePier(){return unavailable()}
  function addStorageRoot(){return unavailable()}
  function enrollDiscovered(){return unavailable()}
  function previewEpochs(){return unavailable()}
  function previewExternalPier(){return unavailable()}
  function prunePreview(){return unavailable()}
  function recoverImport(){return unavailable()}
  function importExternalPier(){return unavailable()}
  function importFake(){return unavailable()}
  function loadImportSources(){return unavailable()}
  function registerComet(){return unavailable()}
  function registerFake(){return unavailable()}
  function saveRoutine(){return unavailable()}
  function startRemoteSetup(){return unavailable()}
  function resumeRemoteSetup(){return unavailable()}
  function setUpLocal(){return unavailable()}
  function suggestPort(){return unavailable()}
  property QtObject serviceUpgrade:QtObject {
    property bool canStart:false
    property bool busy:false
    property var status:null
    property string message:""
    property string error:""
    function check(){return root.unavailable()}
    function upgrade(){return root.unavailable()}
  }
  property QtObject applicationConnections:QtObject {
    property bool busy:false
    property bool available:false
    property bool fresh:true
    property bool failed:false
    property int snapshotRevision:0
    property string message:"Application sign-in is not connected in this preview."
    property var ships:[]
    property var plugins:[]
    property var attempts:[]
    function refresh(){return false}
  }
  property QtObject projectCreation:QtObject {
    signal finished()
    property bool busy:false
    property bool completed:false
    property string error:""
    property var intent:null
    property var progress:({})
    function start(){return root.unavailable()}
    function resume(){return root.unavailable()}
  }
  property QtObject importRestoration:QtObject {
    property var outcomes:({})
    property var review:null
    property var selection:null
    property var pendingPreview:null
    property string error:""
    function currentReview(){return false}
    function unresolved(){return false}
    function canReviewAgain(){return false}
    function terminal(){return true}
  }
}
