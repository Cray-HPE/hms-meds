@Library('dst-shared@master') _

dockerBuildPipeline {
        githubPushRepo = "Cray-HPE/hms-meds"
        repository = "cray"
        imagePrefix = "cray"
        app = "meds"
        name = "hms-meds"
        description = "Cray mountain endpoint discovery service"
        dockerfile = "Dockerfile"
        slackNotification = ["", "", false, false, true, true]
        product = "csm"
}
