const express = require("express");
const cors = require("cors");
const cookieParser = require("cookie-parser");
const session = require("express-session");

const redisStoreModule = require("connect-redis");
console.log("connect-redis exports:", redisStoreModule);

let RedisStore;
if (redisStoreModule.default) {
  RedisStore = redisStoreModule.default;
} else if (typeof redisStoreModule === "function") {
  RedisStore = redisStoreModule;
} else if (redisStoreModule.RedisStore) {
  RedisStore = redisStoreModule.RedisStore;
} else {
  throw new Error("Cannot find RedisStore in connect-redis module");
}

const { createClient } = require("redis");
const dotenv = require("dotenv");

const factRoutes = require("./routes/factsRoutes");
const dimensionRoutes = require("./routes/dimensionsRoutes");
const cubeRoutes = require("./routes/cubesRoutes");
const authRoutes = require("./routes/authRoutes");

// Load environment variables from .env file (if exists)
dotenv.config();

const app = express();

const redisClient = createClient({
  socket: {
    host: "127.0.0.1",
    port: 6379,
  },
});

redisClient.on("error", (err) => console.log("Redis Client Error", err));

(async () => {
  try {
    await redisClient.connect();
    console.log("Redis client connected");

    const redisStore = new RedisStore({
      client: redisClient,
      prefix: "sess:",
    });

    // Enable CORS - customize options if needed
    app.use(
      cors({
        origin: true,
        credentials: true,
      })
    );

    // Parse JSON bodies
    app.use(express.json());

    // Parse cookies
    app.use(cookieParser());

    // Session middleware using Redis store
    app.use(
      session({
        store: redisStore,
        secret: process.env.SESSION_SECRET || "your-secret-key",
        resave: false,
        saveUninitialized: false,
        cookie: {
          httpOnly: true,
          secure: false,
          sameSite: "lax",
          maxAge: 24 * 60 * 60 * 1000,
        },
      })
    );

    // Mount routes
    app.use("/api/auth", authRoutes);
    app.use("/api/facts", factRoutes);
    app.use("/api/dimensions", dimensionRoutes);
    app.use("/api/cubes", cubeRoutes);

    const PORT = process.env.PORT || 5000;
    app.listen(PORT, () => {
      console.log(`Server running on port ${PORT}`);
    });
  } catch (err) {
    console.error("Could not connect to Redis", err);
  }
})();
