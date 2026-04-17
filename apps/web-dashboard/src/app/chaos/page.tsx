"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Shield, Zap, Clock, AlertTriangle, Server, Database, Wifi } from "lucide-react";

interface Experiment {
  id: string;
  name: string;
  target: string;
  type: string;
  icon: typeof Server;
  description: string;
  status: "idle" | "running" | "completed" | "failed";
  lastRun?: string;
  duration?: string;
}

const initialExperiments: Experiment[] = [
  {
    id: "exp_001", name: "API Latency Injection", target: "API Server", type: "latency",
    icon: Server, description: "Inject 500ms latency on all API endpoints to test client timeout handling.",
    status: "idle", lastRun: "2026-04-13 15:00", duration: "5m",
  },
  {
    id: "exp_002", name: "Database Connection Kill", target: "PostgreSQL", type: "kill",
    icon: Database, description: "Terminate database connections to verify reconnection and circuit breaker logic.",
    status: "idle", lastRun: "2026-04-12 10:30", duration: "2m",
  },
  {
    id: "exp_003", name: "Charging Engine Crash", target: "Charging Engine", type: "crash",
    icon: Zap, description: "Simulate Rust charging engine crash to validate Go fallback and graceful degradation.",
    status: "idle", lastRun: "2026-04-11 09:00", duration: "3m",
  },
  {
    id: "exp_004", name: "Network Partition", target: "AMF Gateway", type: "partition",
    icon: Wifi, description: "Simulate network partition between API server and AMF to test session continuity.",
    status: "idle", duration: "10m",
  },
  {
    id: "exp_005", name: "Redis Cache Flush", target: "Redis", type: "flush",
    icon: Database, description: "Flush all Redis keys to test cache miss handling and cold-start performance.",
    status: "idle", lastRun: "2026-04-10 14:00", duration: "1m",
  },
  {
    id: "exp_006", name: "High Load Simulation", target: "API Server", type: "load",
    icon: AlertTriangle, description: "Send 10,000 concurrent requests to test rate limiting and autoscaling.",
    status: "idle", duration: "15m",
  },
];

const statusVariant = (s: string) =>
  s === "running" ? "warning" : s === "completed" ? "success" : s === "failed" ? "destructive" : "secondary";

export default function ChaosPage() {
  const [experiments, setExperiments] = useState(initialExperiments);

  const runExperiment = (id: string) => {
    setExperiments((prev) =>
      prev.map((e) =>
        e.id === id ? { ...e, status: "running" as const } : e
      )
    );
    // Simulate completion after a delay
    setTimeout(() => {
      setExperiments((prev) =>
        prev.map((e) =>
          e.id === id
            ? { ...e, status: (Math.random() > 0.2 ? "completed" : "failed") as Experiment["status"], lastRun: new Date().toISOString().slice(0, 16).replace("T", " ") }
            : e
        )
      );
    }, 3000);
  };

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center gap-3">
        <Shield className="size-8 text-primary" />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Chaos Engineering</h1>
          <p className="text-muted-foreground mt-1">Run controlled failure experiments to validate system resilience.</p>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card size="sm">
          <CardContent className="pt-4">
            <div className="text-2xl font-bold">{experiments.filter((e) => e.status === "completed").length}</div>
            <p className="text-xs text-muted-foreground">Experiments Passed</p>
          </CardContent>
        </Card>
        <Card size="sm">
          <CardContent className="pt-4">
            <div className="text-2xl font-bold">{experiments.filter((e) => e.status === "failed").length}</div>
            <p className="text-xs text-muted-foreground">Experiments Failed</p>
          </CardContent>
        </Card>
        <Card size="sm">
          <CardContent className="pt-4">
            <div className="text-2xl font-bold">{experiments.filter((e) => e.status === "running").length}</div>
            <p className="text-xs text-muted-foreground">Currently Running</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {experiments.map((exp) => (
          <Card key={exp.id}>
            <CardHeader className="flex-row items-start justify-between pb-2">
              <div className="flex items-center gap-3">
                <div className="rounded-lg bg-muted p-2">
                  <exp.icon className="size-5 text-muted-foreground" />
                </div>
                <div>
                  <CardTitle className="text-base">{exp.name}</CardTitle>
                  <CardDescription className="text-xs">Target: {exp.target}</CardDescription>
                </div>
              </div>
              <Badge variant={statusVariant(exp.status)}>{exp.status}</Badge>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground mb-4">{exp.description}</p>
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1"><Clock className="size-3" />{exp.duration}</span>
                  {exp.lastRun && <span>Last: {exp.lastRun}</span>}
                </div>
                <Button
                  size="sm"
                  variant={exp.status === "running" ? "outline" : "default"}
                  disabled={exp.status === "running"}
                  onClick={() => runExperiment(exp.id)}
                >
                  {exp.status === "running" ? "Running…" : "Run Experiment"}
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
