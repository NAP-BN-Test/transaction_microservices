const axios = require('axios');

const ORCHESTRATOR_URL = 'http://localhost:8081';

async function testOrchestration() {
  console.log('=== Testing Orchestration Pattern ===\n');

  try {
    // Test 1: Success Case
    console.log('1. Testing Success Case...');
    const successResponse = await axios.post(`${ORCHESTRATOR_URL}/orchestrate/order`, {
      customer_id: 1,
      items: [
        { product_id: 1, quantity: 2 },
        { product_id: 2, quantity: 1 }
      ],
      voucher_code: 'SAVE10'
    });

    console.log('Success Response:', successResponse.data);
    const workflowId = successResponse.data.workflow_id;

    // Wait and check status
    console.log('\n2. Checking workflow status...');
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    const statusResponse = await axios.get(`${ORCHESTRATOR_URL}/orchestrate/${workflowId}`);
    console.log('Workflow Status:', JSON.stringify(statusResponse.data, null, 2));

    // Test 2: Failure Case
    console.log('\n3. Testing Failure Case...');
    const failResponse = await axios.post(`${ORCHESTRATOR_URL}/orchestrate/order`, {
      customer_id: 1,
      items: [
        { product_id: 999, quantity: 1 } // Invalid product
      ]
    });

    console.log('Fail Response:', failResponse.data);
    const failWorkflowId = failResponse.data.workflow_id;

    // Wait and check failed status
    console.log('\n4. Checking failed workflow status...');
    await new Promise(resolve => setTimeout(resolve, 3000));
    
    const failStatusResponse = await axios.get(`${ORCHESTRATOR_URL}/orchestrate/${failWorkflowId}`);
    console.log('Failed Workflow Status:', JSON.stringify(failStatusResponse.data, null, 2));

    // Test 3: Manual Compensation
    console.log('\n5. Testing manual compensation...');
    try {
      const compensateResponse = await axios.post(`${ORCHESTRATOR_URL}/orchestrate/${failWorkflowId}/compensate`);
      console.log('Compensation Response:', compensateResponse.data);
    } catch (error) {
      console.log('Compensation Error:', error.response?.data || error.message);
    }

    // Test 4: Stock Failure
    console.log('\n6. Testing stock failure...');
    const stockFailResponse = await axios.post(`${ORCHESTRATOR_URL}/orchestrate/order`, {
      customer_id: 2,
      items: [
        { product_id: 1, quantity: 1000 } // Too much quantity
      ]
    });

    console.log('Stock Fail Response:', stockFailResponse.data);
    const stockFailWorkflowId = stockFailResponse.data.workflow_id;

    await new Promise(resolve => setTimeout(resolve, 3000));
    const stockFailStatusResponse = await axios.get(`${ORCHESTRATOR_URL}/orchestrate/${stockFailWorkflowId}`);
    console.log('Stock Fail Workflow Status:', JSON.stringify(stockFailStatusResponse.data, null, 2));

  } catch (error) {
    console.error('Test Error:', error.response?.data || error.message);
  }
}

async function comparePatterns() {
  console.log('\n=== Comparing Saga vs Orchestration ===\n');
  
  console.log('SAGA PATTERN:');
  console.log('- Decentralized coordination');
  console.log('- Services communicate via events');
  console.log('- Each service handles its own compensation');
  console.log('- Better for loose coupling');
  console.log('- Harder to track overall workflow state');
  
  console.log('\nORCHESTRATION PATTERN:');
  console.log('- Centralized coordination');
  console.log('- Orchestrator calls services directly');
  console.log('- Orchestrator handles compensation logic');
  console.log('- Easier to track and debug workflow');
  console.log('- Single point of failure');
  
  console.log('\nUSE CASES:');
  console.log('- Saga: Complex business domains, microservices autonomy');
  console.log('- Orchestration: Simple workflows, centralized control needed');
}

// Run tests
if (require.main === module) {
  testOrchestration()
    .then(() => comparePatterns())
    .then(() => console.log('\n=== Tests Completed ==='))
    .catch(console.error);
}

module.exports = { testOrchestration, comparePatterns };