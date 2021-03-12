@Library('mdblp-library') _
def builderImage
pipeline {
    agent any
    stages {
        stage('Initialization') {
            steps {
                script {
                    utils.initPipeline()
                    if(env.GIT_COMMIT == null) {
                        // git commit id must be a 40 characters length string (lower case or digits)
                        env.GIT_COMMIT = "f".multiply(40)
                    }
                    builderImage = docker.build('go-build-image','-f ./Dockerfile.build .')
                    env.RUN_ID = UUID.randomUUID().toString()
                    docker.image('docker.ci.diabeloop.eu/ci-toolbox').inside() {
                        env.version = sh (
                            script: 'release-helper get-version',
                            returnStdout: true
                        ).trim().toUpperCase()
                    }
                }
            }
        }
        stage('Build ') {
            steps {
                script {
                    builderImage.inside("") {
                        sh "$WORKSPACE/build.sh"
                    }
                }
            }
        }
        stage('Test ') {
            steps {
                echo 'start mongo to serve as a testing db'
                sh 'docker network create twtest${RUN_ID} && docker run --rm -d --net=twtest${RUN_ID} --name=mongo4twtest${RUN_ID} mongo:4.2'
                script {
                    builderImage.inside("--net=twtest${RUN_ID}") {
                        sh "TIDEPOOL_STORE_ADDRESSES=mongo4twtest${RUN_ID}:27017 TIDEPOOL_STORE_DATABASE=data_test $WORKSPACE/test.sh"
                    }
                }
            }
            post {
                always {
                    sh 'docker stop mongo4twtest${RUN_ID} && docker network rm twtest${RUN_ID}'
                }
            }
        }
        stage('Package') {
            steps {
                pack()
            }
        }
        stage('Documentation') {
            steps {
                script {
                    builderImage.inside("") {
                        sh """
                            export TRAVIS_TAG=${version}
                            ./buildDoc.sh
                            ./buildSoup.sh
                        """              
                        stash name: "doc", includes: "docs/*"
                    }
                }
                
            }
        }
        stage('Publish') {
            when { branch "dblp" }
            steps {
                publish()
            }
        }
    }
}
