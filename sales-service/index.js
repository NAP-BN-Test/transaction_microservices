require('dotenv').config();

const express = require('express');
const { Pool } = require('pg');
const redis = require('redis');
const { v4: uuidv4 } = require('uuid');

const app = express();
app.use(express.json());

// Database connection
const pool = new Pool({
  host: process.env.DB_HOST,
  port: process.env.DB_PORT,
  user: process.env.DB_USER,
  password: process.env.DB_PASSWORD,
  database: process.env.DB_NAME,
});

// Redis connection
const redisClient = redis.createClient({
  url: `redis://${process.env.REDIS_URL}`
});

redisClient.connect();

// Routes
app.get('/vouchers', getVouchers);
app.post('/vouchers', createVoucher);
app.post('/sales/process', processSalesTransaction);
app.get('/sales/:orderId', getSalesTransaction);

// Saga event processing
processSagaEvents();
// Outbox processor
processOutboxEvents();

app.listen(3000, () => {
  console.log('Sales service running on port 3000');
});

async function getVouchers(req, res) {
  try {
    const result = await pool.query('SELECT * FROM vouchers WHERE is_active = true');
    res.json(result.rows);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
}

async function createVoucher(req, res) {
  const { code, discount_percent, max_discount } = req.body;
  
  try {
    const result = await pool.query(
      'INSERT INTO vouchers (code, discount_percent, max_discount) VALUES ($1, $2, $3) RETURNING *',
      [code, discount_percent, max_discount]
    );
    res.status(201).json(result.rows[0]);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
}

async function processSalesTransaction(req, res) {
  const { order_id, customer_id, original_amount, voucher_code } = req.body;
  
  const client = await pool.connect();
  
  try {
    await client.query('BEGIN');
    
    let discount_amount = 0;
    let voucher_id = null;
    
    // Apply voucher if provided
    if (voucher_code) {
      const voucherResult = await client.query(
        'SELECT * FROM vouchers WHERE code = $1 AND is_active = true',
        [voucher_code]
      );
      
      if (voucherResult.rows.length > 0) {
        const voucher = voucherResult.rows[0];
        voucher_id = voucher.id;
        
        discount_amount = (original_amount * voucher.discount_percent) / 100;
        if (voucher.max_discount && discount_amount > voucher.max_discount) {
          discount_amount = voucher.max_discount;
        }
      }
    }
    
    const final_amount = original_amount - discount_amount;
    
    // Create sales transaction
    const result = await client.query(
      `INSERT INTO sales_transactions 
       (order_id, customer_id, voucher_id, original_amount, discount_amount, final_amount, status) 
       VALUES ($1, $2, $3, $4, $5, $6, 'completed') RETURNING *`,
      [order_id, customer_id, voucher_id, original_amount, discount_amount, final_amount]
    );
    
    // Store event in outbox within same transaction
    const sagaEvent = {
      id: uuidv4(),
      type: 'SALES_TRANSACTION_COMPLETED',
      order_id: order_id,
      data: {
        transaction_id: result.rows[0].id,
        final_amount: final_amount,
        discount_amount: discount_amount
      },
      timestamp: new Date()
    };
    
    await client.query(
      'INSERT INTO outbox_events (event_id, event_type, aggregate_id, event_data) VALUES ($1, $2, $3, $4)',
      [sagaEvent.id, sagaEvent.type, order_id, JSON.stringify(sagaEvent.data)]
    );
    
    await client.query('COMMIT');
    
    res.json(result.rows[0]);
    
  } catch (error) {
    await client.query('ROLLBACK');
    
    // Store failure event in outbox (separate transaction)
    const failureClient = await pool.connect();
    try {
      const sagaEvent = {
        id: uuidv4(),
        type: 'SALES_TRANSACTION_FAILED',
        order_id: order_id,
        data: { error: error.message },
        timestamp: new Date()
      };
      
      await failureClient.query(
        'INSERT INTO outbox_events (event_id, event_type, aggregate_id, event_data) VALUES ($1, $2, $3, $4)',
        [sagaEvent.id, sagaEvent.type, order_id, JSON.stringify(sagaEvent.data)]
      );
    } finally {
      failureClient.release();
    }
    
    res.status(500).json({ error: error.message });
  } finally {
    client.release();
  }
}

async function getSalesTransaction(req, res) {
  const { orderId } = req.params;
  
  try {
    const result = await pool.query(
      `SELECT st.*, v.code as voucher_code 
       FROM sales_transactions st 
       LEFT JOIN vouchers v ON st.voucher_id = v.id 
       WHERE st.order_id = $1`,
      [orderId]
    );
    
    if (result.rows.length === 0) {
      return res.status(404).json({ error: 'Sales transaction not found' });
    }
    
    res.json(result.rows[0]);
  } catch (error) {
    res.status(500).json({ error: error.message });
  }
}

async function publishSagaEvent(event) {
  await redisClient.publish('saga_responses', JSON.stringify(event));
}

async function processSagaEvents() {
  const subscriber = redisClient.duplicate();
  await subscriber.connect();
  
  await subscriber.subscribe('saga_events', async (message) => {
    try {
      const event = JSON.parse(message);
      
      switch (event.type) {
        case 'ORDER_CREATED':
          await handleOrderCreated(event);
          break;
      }
    } catch (error) {
      console.error('Error processing saga event:', error);
    }
  });
}

async function handleOrderCreated(event) {
  try {
    // Auto-process sales transaction for the order
    const { customer_id, total_amount } = event.data;
    
    const client = await pool.connect();
    
    try {
      await client.query('BEGIN');
      
      // Create sales transaction without voucher (can be updated later)
      const result = await client.query(
        `INSERT INTO sales_transactions 
         (order_id, customer_id, original_amount, discount_amount, final_amount, status) 
         VALUES ($1, $2, $3, 0, $3, 'pending') RETURNING *`,
        [event.order_id, customer_id, total_amount]
      );
      
      // Store success event in outbox within same transaction
      const sagaEvent = {
        id: uuidv4(),
        type: 'SALES_TRANSACTION_COMPLETED',
        order_id: event.order_id,
        data: {
          transaction_id: result.rows[0].id,
          final_amount: total_amount,
          discount_amount: 0
        },
        timestamp: new Date()
      };
      
      await client.query(
        'INSERT INTO outbox_events (event_id, event_type, aggregate_id, event_data) VALUES ($1, $2, $3, $4)',
        [sagaEvent.id, sagaEvent.type, event.order_id, JSON.stringify(sagaEvent.data)]
      );
      
      await client.query('COMMIT');
      
    } catch (error) {
      await client.query('ROLLBACK');
      
      // Store failure event in outbox (separate transaction)
      const failureClient = await pool.connect();
      try {
        const sagaEvent = {
          id: uuidv4(),
          type: 'SALES_TRANSACTION_FAILED',
          order_id: event.order_id,
          data: { error: error.message },
          timestamp: new Date()
        };
        
        await failureClient.query(
          'INSERT INTO outbox_events (event_id, event_type, aggregate_id, event_data) VALUES ($1, $2, $3, $4)',
          [sagaEvent.id, sagaEvent.type, event.order_id, JSON.stringify(sagaEvent.data)]
        );
      } finally {
        failureClient.release();
      }
    } finally {
      client.release();
    }
    
  } catch (error) {
    console.error('Error handling order created event:', error);
  }
}

async function processOutboxEvents() {
  setInterval(async () => {
    try {
      const result = await pool.query(
        'SELECT id, event_id, event_type, aggregate_id, event_data FROM outbox_events WHERE processed = false ORDER BY created_at'
      );
      
      for (const row of result.rows) {
        const sagaEvent = {
          id: row.event_id,
          type: row.event_type,
          order_id: row.aggregate_id,
          data: typeof row.event_data === 'string' ? JSON.parse(row.event_data) : row.event_data,
          timestamp: new Date()
        };
        
        // Publish to Redis
        await publishSagaEvent(sagaEvent);
        
        // Mark as processed
        await pool.query('UPDATE outbox_events SET processed = true WHERE id = $1', [row.id]);
      }
    } catch (error) {
      console.error('Error processing outbox events:', error);
    }
  }, 2000);
}