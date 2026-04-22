use crate::api::proto::{CpuInfo, DiskInfo, MemoryInfo, RealtimeResource, SystemOverview};
use sysinfo::{Disks, System};

pub struct SystemInfoService;

impl SystemInfoService {
    pub fn new() -> Self {
        Self
    }

    pub fn get_overview(&self) -> SystemOverview {
        let mut system = System::new_all();
        system.refresh_all();

        let hostname = System::host_name().unwrap_or_else(|| "unknown".to_string());
        let os = System::long_os_version().unwrap_or_else(|| "unknown".to_string());
        let kernel = System::kernel_version().unwrap_or_else(|| "unknown".to_string());
        let uptime_seconds = System::uptime();
        let uptime = format!("{}s", uptime_seconds);

        let cpus = system.cpus();
        let logical_cores = cpus.len() as u32;
        let cpu_model = cpus
            .first()
            .map(|cpu| cpu.brand().to_string())
            .unwrap_or_else(|| "unknown".to_string());
        let cpu_usage_percent = if cpus.is_empty() {
            0.0
        } else {
            cpus.iter().map(|cpu| cpu.cpu_usage() as f64).sum::<f64>() / cpus.len() as f64
        };

        let total_memory = system.total_memory();
        let used_memory = system.used_memory();
        let memory_usage_percent = if total_memory == 0 {
            0.0
        } else {
            used_memory as f64 * 100.0 / total_memory as f64
        };

        let disks = Disks::new_with_refreshed_list();
        let disk_items: Vec<DiskInfo> = disks
            .iter()
            .map(|disk| {
                let total = disk.total_space();
                let used = total.saturating_sub(disk.available_space());
                let usage = if total == 0 {
                    0.0
                } else {
                    used as f64 * 100.0 / total as f64
                };
                DiskInfo {
                    mount_point: disk.mount_point().to_string_lossy().into_owned(),
                    total_bytes: total,
                    used_bytes: used,
                    usage_percent: usage,
                }
            })
            .collect();

        SystemOverview {
            hostname,
            os,
            kernel,
            uptime,
            cpu: Some(CpuInfo {
                model: cpu_model,
                logical_cores,
                usage_percent: cpu_usage_percent,
            }),
            memory: Some(MemoryInfo {
                total_bytes: total_memory,
                used_bytes: used_memory,
                usage_percent: memory_usage_percent,
            }),
            disks: disk_items,
        }
    }

    pub fn get_realtime_resource(&self) -> RealtimeResource {
        let mut system = System::new_all();
        system.refresh_all();

        let cpus = system.cpus();
        let cpu_usage_percent = if cpus.is_empty() {
            0.0
        } else {
            cpus.iter().map(|cpu| cpu.cpu_usage() as f64).sum::<f64>() / cpus.len() as f64
        };

        let total_memory = system.total_memory();
        let used_memory = system.used_memory();
        let memory_usage_percent = if total_memory == 0 {
            0.0
        } else {
            used_memory as f64 * 100.0 / total_memory as f64
        };

        let disks = Disks::new_with_refreshed_list();
        let (disk_used, disk_total) = disks.iter().fold((0u64, 0u64), |(used, total), disk| {
            let size = disk.total_space();
            let used_now = size.saturating_sub(disk.available_space());
            (used.saturating_add(used_now), total.saturating_add(size))
        });
        let disk_usage_percent = if disk_total == 0 {
            0.0
        } else {
            disk_used as f64 * 100.0 / disk_total as f64
        };

        let load_avg = System::load_average();
        RealtimeResource {
            cpu_usage_percent,
            memory_usage_percent,
            disk_usage_percent,
            load_average_1m: load_avg.one,
            load_average_5m: load_avg.five,
            load_average_15m: load_avg.fifteen,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::SystemInfoService;

    #[test]
    fn overview_contains_basic_fields() {
        let service = SystemInfoService::new();
        let overview = service.get_overview();

        assert!(
            !overview.hostname.trim().is_empty(),
            "hostname should not be empty"
        );
        assert!(
            !overview.os.trim().is_empty(),
            "os description should not be empty"
        );
        assert!(
            !overview.kernel.trim().is_empty(),
            "kernel version should not be empty"
        );
        assert!(
            overview.cpu.is_some(),
            "cpu information should be returned"
        );
        assert!(
            overview.memory.is_some(),
            "memory information should be returned"
        );
    }

    #[test]
    fn realtime_resource_contains_non_negative_numbers() {
        let service = SystemInfoService::new();
        let resource = service.get_realtime_resource();

        assert!(
            resource.cpu_usage_percent.is_finite(),
            "cpu usage should be finite"
        );
        assert!(
            resource.memory_usage_percent.is_finite(),
            "memory usage should be finite"
        );
        assert!(
            resource.disk_usage_percent.is_finite(),
            "disk usage should be finite"
        );
        assert!(
            resource.cpu_usage_percent >= 0.0,
            "cpu usage should be non-negative"
        );
        assert!(
            resource.memory_usage_percent >= 0.0,
            "memory usage should be non-negative"
        );
        assert!(
            resource.disk_usage_percent >= 0.0,
            "disk usage should be non-negative"
        );
    }
}
