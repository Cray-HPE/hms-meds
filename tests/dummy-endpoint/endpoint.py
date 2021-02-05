#  MIT License
#
#  (C) Copyright [2019-2021] Hewlett Packard Enterprise Development LP
#
#  Permission is hereby granted, free of charge, to any person obtaining a
#  copy of this software and associated documentation files (the "Software"),
#  to deal in the Software without restriction, including without limitation
#  the rights to use, copy, modify, merge, publish, distribute, sublicense,
#  and/or sell copies of the Software, and to permit persons to whom the
#  Software is furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included
#  in all copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
#  THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
#  OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
#  ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
#  OTHER DEALINGS IN THE SOFTWARE.

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