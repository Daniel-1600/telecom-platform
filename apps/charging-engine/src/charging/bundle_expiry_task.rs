use tokio::time::{interval, Duration};
use tracing::{error, info};
use super::bundle_repo::BundleRepo;

/// Spawns a background task that expires stale subscriber bundles every 5 minutes.
/// Must be called once at engine startup.
pub fn spawn_bundle_expiry_task(bundle_repo: BundleRepo) {
    tokio::spawn(async move {
        let mut ticker = interval(Duration::from_secs(300)); // every 5 minutes
        loop {
            ticker.tick().await;
            match bundle_repo.expire_stale_bundles().await {
                Ok(count) if count > 0 => {
                    info!(expired_count = count, "Expired stale subscriber bundles");
                }
                Ok(_) => {}
                Err(e) => {
                    error!(error = %e, "Failed to expire stale bundles");
                }
            }
        }
    });
}
