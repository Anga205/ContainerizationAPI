public class Main {
    public static void main(String[] args) throws Exception {
        new ProcessBuilder("/bin/sh", "-c", "exec -a orphanjavagc sleep 60").start();
    }
}
