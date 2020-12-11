@Library('dst-shared@release/shasta-1.4') _

dockerBuildPipeline {
        repository = "cray"
        imagePrefix = "cray"
        app = "meds"
        name = "hms-meds"
        description = "Cray mountain endpoint discovery service"
        dockerfile = "Dockerfile"
        slackNotification = ["", "", false, false, true, true]
        product = "csm"
}
