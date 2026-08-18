// k6 load test for the core booking flow.
//
// Run against a local/dev server:
//   k6 run scripts/loadtest/booking-flow.js
// Or via Docker (no local install needed):
//   docker run --rm -i -v "$PWD/scripts/loadtest:/scripts" \
//     grafana/k6 run /scripts/booking-flow.js
//
// Environment overrides:
//   BASE_URL   (default http://localhost:9090/api/v1)
//   ADMIN_EMAIL / ADMIN_PASSWORD (defaults match the local dev bootstrap)
//   VUS, DURATION (k6 standard)
import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:9090/api/v1';
const ADMIN_EMAIL = __ENV.ADMIN_EMAIL || 'admin@laundrypro.local';
const ADMIN_PASSWORD = __ENV.ADMIN_PASSWORD || 'Admin@12345';

export const options = {
  stages: [
    { duration: '30s', target: 10 }, // ramp up
    { duration: '1m', target: 10 },  // steady
    { duration: '30s', target: 0 },  // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
    http_req_failed: ['rate<0.01'],   // <1% errors
  },
};

// One shared token per VU (login is expensive: bcrypt cost 12), plus real
// customer/service-type ids so the booking write passes validation.
export function setup() {
  const res = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    email: ADMIN_EMAIL,
    password: ADMIN_PASSWORD,
  }), { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'login 200': (r) => r.status === 200 });
  const token = res.json('data.access_token');
  const headers = { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };

  const me = http.post(`${BASE_URL}/auth/me`, '{}', { headers });
  const customerID = me.json('data.id') || '';
  const st = http.post(`${BASE_URL}/service-types/list`,
    JSON.stringify({ limit: 1 }), { headers });
  const serviceTypeID = st.json('data.0.id') || '';

  return { token, customerID, serviceTypeID };
}

export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${data.token}`,
  };

  // Read path: list service types (cursor pagination, filters).
  const list = http.post(`${BASE_URL}/service-types/list`,
    JSON.stringify({ limit: 20, filters: {} }), { headers });
  check(list, { 'service-types/list 200': (r) => r.status === 200 });

  // Write path: create a booking (enqueues an outbox event -> saga step).
  // Requires customer_id + service_type_id; strictly expect 201.
  const booking = http.post(`${BASE_URL}/bookings`, JSON.stringify({
    customer_id: data.customerID,
    service_type_id: data.serviceTypeID,
    pickup_address: '1 Load Test Ave',
    estimated_amount: 120,
    currency: 'INR',
  }), { headers });
  check(booking, { 'booking create 201': (r) => r.status === 201 });

  // Follow-up read: paged bookings list with a date-range filter.
  const bookings = http.post(`${BASE_URL}/bookings/list`,
    JSON.stringify({ limit: 10, filters: { created_at: 'gte:' + new Date(Date.now() - 86400000).toISOString() } }),
    { headers });
  check(bookings, { 'bookings/list 200': (r) => r.status === 200 });

  sleep(1);
}
