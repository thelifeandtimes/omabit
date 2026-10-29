# Corrected gateway run interrupted before a result

Appliance25 contains the M402 tmpfs correction, the current runner and the same
sealed generation07 inputs. Compilation, armed host refusal and vet pass. A fresh
no-NIC VM (run02) passed offline setup, engine and Vere checks and entered managed
gateway initialization. Its launch session then ended with status143. The exact
VM scope was confirmed inactive. Neither the launcher nor the guest produced a
final result, and the serial log contains no clean shutdown. The termination
cause is unknown; do not attribute it to the gateway, OOM or a tool deadline.

Independent read-only audit04 mounted ext4 with ro,noload and refused at the
database-fully-checkpointed gate. It did not replay the database journal or
reinterpret the incomplete run as a success. The audit itself shut down cleanly;
that does not mean the original runtime did. Both base and overlay hashes are
unchanged. The private disk is retained with compatibility path and SHA256
11229f65bd631539cf6131eb35f495726658303628409f8d7f8e8031a7d57cfe.

M403 supersedes the running/pending statement for run02 only. M402's source checks
remain valid; current managed runtime/recovery qualification remains open. The
next attempt uses a fresh disk with the same immutable appliance, under a unique
bounded transient supervisor independent of the interactive command session.
Keep the existing inner VM limits, no-NIC boundary, no-retry rule and resource
admission floors. No protected ship, installed plugin or production service was
accessed. The interrupted disk is not reused for another initialization attempt.
