public class Main {
    public static void main(String[] args) throws Exception {
        while (true) {
            new ProcessBuilder("/bin/sh", "-c", "sleep 1000").start();
        }
    }
}
