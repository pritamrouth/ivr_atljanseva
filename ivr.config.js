module.exports = {
  apps: [
    {
      name: "Atal Janseva IVR",
      script: "build/ivr_v.01", // Point to the compiled binary
      env: { PORT: 5000 }
    }
  ]
};