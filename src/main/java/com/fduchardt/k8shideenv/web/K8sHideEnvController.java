package com.fduchardt.k8shideenv.web;

import com.fduchardt.k8shideenv.service.*;
import lombok.*;
import lombok.extern.slf4j.*;
import org.springframework.web.bind.annotation.*;

@RequiredArgsConstructor
@RestController
@Slf4j
public class K8sHideEnvController {

    private final K8sHideEnvService k8sHideEnvService;

    @PostMapping(path="/mutate")
    public String mutate(@RequestBody String admissionReview) {
        log.info(admissionReview);
        String admissionReturn = k8sHideEnvService.createAdmissionResponse(admissionReview);
        log.info(admissionReturn);
        return admissionReturn;
    }
}
