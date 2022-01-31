const exec = require('child_process').exec;
const data = require('../fixtures/automation/automationTestData.js');

function setupAnsible(){
    const setupAnsibleTemplate = exec(`sh ./cypress/scripts/setupAnsibleTemplate.sh \
                                            ${data.ansibleTestData.projectName} \
                                            ${data.ansibleTestData.templateName} \
                                            ${data.ansibleTestData.templateRepo}`);
    setupAnsibleTemplate.stdout.on('data', (data)=>{
        console.log(data); 
    });
    setupAnsibleTemplate.stderr.on('data', (data)=>{
        console.error(data);
    });
}

setupAnsible();
