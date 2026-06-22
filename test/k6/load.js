import http from "k6/http";
import { check, sleep } from "k6";

// Light load/benchmark profile for the Healthwatch API: ramps up to a
// modest steady load, holds it, then ramps down. Tune stages and
// thresholds to taste once you know your target hardware.
export const options = {
  stages: [
    { duration: "10s", target: 10 },
    { duration: "30s", target: 10 },
    { duration: "10s", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<200"],
  },
};

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export default function () {
  const responses = http.batch([
    ["GET", `${BASE_URL}/healthz`],
    ["GET", `${BASE_URL}/api/v1/checks`],
  ]);

  for (const res of responses) {
    check(res, { "status is 200": (r) => r.status === 200 });
  }

  sleep(1);
}
