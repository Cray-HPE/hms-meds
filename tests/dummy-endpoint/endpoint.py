from flask import Flask
app = Flask(__name__)

@app.route('/redfish/v1/')
@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def hello_world():
    return """{
        "@odata.id":"/redfish/v1",
        "@odata.type":"#ServiceRoot.v1_6_0.ServiceRoot",
        "AccountService":{
            "@odata.id":"/redfish/v1/AccountService"
        },
        "Chassis":{
            "@odata.id":"/redfish/v1/Chassis"
        },
        "Description":"The Redfish ServiceRoot",
        "JsonSchemas":{
            "@odata.id":"/redfish/v1/JsonSchemas"
        },
        "Managers":{
            "@odata.id":"/redfish/v1/Managers"
        },
        "Name":"Service Root",
        "RedfishVersion":"2019.1",
        "Systems":{
            "@odata.id":"/redfish/v1/Systems"
        },
        "UpdateService":{
            "@odata.id":"/redfish/v1/UpdateService"
        }
    }"""

app.run(host="0.0.0.0", port=443, ssl_context='adhoc')