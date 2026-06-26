import http from "k6/http";
import { check, sleep } from "k6";

// 10 VUs for 15s, as required - tune further once you know the target
// hardware.
export const options = {
  vus: 10,
  duration: "15s",
  thresholds: {
    http_req_duration: ["p(95)<200"],
    http_req_failed: ["rate<0.01"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const health = http.get(`${BASE_URL}/health`);
  check(health, { "health is 200": (r) => r.status === 200 });

  const list = http.get(`${BASE_URL}/items`);
  check(list, { "list items is 200": (r) => r.status === 200 });

  const created = http.post(
    `${BASE_URL}/items`,
    JSON.stringify({
      name: `bench-item-${__VU}-${__ITER}-${Date.now()}`,
      url: "https://example.com",
    }),
    { headers: { "Content-Type": "application/json" } },
  );
  check(created, { "create item is 201": (r) => r.status === 201 });

  sleep(1);
}
