package com.fduchardt.k8shideenv;

import com.fduchardt.k8shideenv.service.*;
import lombok.extern.slf4j.*;
import org.springframework.beans.factory.annotation.*;
import org.springframework.boot.*;
import org.springframework.boot.autoconfigure.*;

@SpringBootApplication
public class K8sHideEnvApplication {

    @Autowired
    K8sHideEnvService k8sDispatcher;

    public static void main(String[] args) {
        SpringApplication.run(K8sHideEnvApplication.class, args);
    }
}
